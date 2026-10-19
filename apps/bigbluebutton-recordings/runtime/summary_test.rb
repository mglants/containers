# frozen_string_literal: true

require 'tmpdir'
require 'socket'
require '/usr/local/bigbluebutton/core/lib/recordandplayback/summary'

class SummaryTest
  def assert(condition, message = 'Summary regression check failed')
    raise message unless condition
  end
  RESULT = {
    'transcription' => { 'full_transcript' => 'We agreed to test Russian and English. <script>alert(1)</script>' },
    'summarization' => { 'success' => true, 'results' => 'Test the launch & document the result.' }
  }.freeze

  class Provider
    attr_reader :calls
    def initialize
      @calls = 0
    end
    def result(_audio, _state)
      @calls += 1
      RESULT
    end
  end

  class ResumingGladia < BigBlueButton::Summary::Gladia
    attr_reader :requests
    def initialize
      super(key: 'test-only')
      @requests = []
    end
    def request(path, **_options)
      @requests << path
      { 'status' => 'done', 'result' => RESULT }
    end
  end

  def test_resume_existing_job_and_cache_result_without_resubmitting_audio
    Dir.mktmpdir do |state|
      BigBlueButton::Summary.save(File.join(state, 'job.json'), { 'id' => 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee' })
      client = ResumingGladia.new
      2.times { assert RESULT == client.result('/missing-audio', state) }
      assert client.requests == ['/v2/pre-recorded/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee']
      assert File.stat(File.join(state, 'result.json')).mode & 0777 == 0600
    end
  end

  def test_ambiguous_submission_never_creates_another_job
    Dir.mktmpdir do |state|
      BigBlueButton::Summary.save(File.join(state, 'submission-started.json'), {})
      client = ResumingGladia.new
      rejected = false
      begin
        client.result('/missing-audio', state)
      rescue BigBlueButton::Summary::Error
        rejected = true
      end
      assert rejected, 'Interrupted submission must require inspection before another POST'
      assert client.requests.empty?
    end
  end

  class FakeOpenAI < BigBlueButton::Summary::OpenAI
    attr_reader :requests
    def initialize
      @requests = []
    end
    def request(path, body: nil, audio: nil)
      @requests << [path, body, audio]
      if path == 'audio/transcriptions'
        raise 'Expected a real audio segment' unless File.size?(audio)
        { 'text' => 'Alice will test the deployment tomorrow.' }
      else
        { 'choices' => [{ 'message' => { 'content' => 'Alice: test deployment tomorrow.' } }] }
      end
    end
  end

  def test_openai_segments_audio_summarizes_and_resumes_without_new_requests
    Dir.mktmpdir do |state|
      audio = File.join(state, 'source.ogg')
      _, _, status = Open3.capture3('ffmpeg', '-v', 'error', '-f', 'lavfi', '-i', 'sine=frequency=440:duration=1', '-c:a', 'libopus', audio)
      assert status.success?
      client = FakeOpenAI.new
      result = client.result(audio, state)
      assert result['transcription']['full_transcript'].include?('Alice')
      assert result['summarization']['success']
      assert client.requests.map(&:first) == %w[audio/transcriptions chat/completions]
      assert client.requests.last[1][:messages].last[:content].include?('Alice')
      assert client.result('/missing-audio', state) == result
      assert client.requests.length == 2
      assert File.stat(File.join(state, 'openai', 'result.json')).mode & 0777 == 0600
    end
  end

  def test_openai_compatible_http_upload_uses_whisper_model_and_bearer_auth
    old = ENV.to_h.slice('OPENAI_BASE_URL', 'OPENAI_API_KEY', 'OPENAI_STT_MODEL')
    server = TCPServer.new('127.0.0.1', 0)
    ENV['OPENAI_BASE_URL'] = "http://127.0.0.1:#{server.addr[1]}"
    ENV['OPENAI_API_KEY'] = 'synthetic-test-key'
    ENV['OPENAI_STT_MODEL'] = 'whisper-1'
    received = Queue.new
    worker = Thread.new do
      socket = server.accept
      headers = +''
      headers << socket.read(1) until headers.end_with?("\r\n\r\n")
      size = headers[/Content-Length: (\d+)/i, 1].to_i
      body = socket.read(size)
      received << [headers, body]
      payload = JSON.generate('text' => 'Synthetic transcription')
      socket.write("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: #{payload.bytesize}\r\nConnection: close\r\n\r\n#{payload}")
      socket.close
    end
    Dir.mktmpdir do |root|
      audio = File.join(root, 'test.mp3')
      File.write(audio, 'synthetic audio bytes')
      result = BigBlueButton::Summary::OpenAI.new.request('audio/transcriptions', audio: audio)
      assert result['text'] == 'Synthetic transcription'
    end
    worker.join
    headers, body = received.pop
    assert headers.start_with?('POST /v1/audio/transcriptions HTTP/1.1')
    assert headers.include?('Authorization: Bearer synthetic-test-key')
    assert body.include?('whisper-1')
    assert body.include?('name="file"')
    assert body.include?('synthetic audio bytes')
  ensure
    worker&.kill
    server&.close
    %w[OPENAI_BASE_URL OPENAI_API_KEY OPENAI_STT_MODEL].each { |name| old.key?(name) ? ENV[name] = old[name] : ENV.delete(name) }
  end

  def test_provider_selection_uses_stt_environment_and_rejects_unknown_provider
    old = ENV.to_h.slice('STT_PROVIDER', 'GLADIA_API_KEY', 'OPENAI_API_KEY')
    ENV['GLADIA_API_KEY'] = 'test-only'
    ENV['OPENAI_API_KEY'] = 'test-only'
    %w[gladia openai].each do |name|
      ENV['STT_PROVIDER'] = name.upcase
      assert BigBlueButton::Summary.provider.class.name.split('::').last.downcase == name
    end
    ENV['STT_PROVIDER'] = 'unsupported'
    begin
      BigBlueButton::Summary.provider
      raise 'Unknown provider must be rejected'
    rescue BigBlueButton::Summary::Error
      # Expected, without accidentally sending audio to a fallback provider.
    end
  ensure
    %w[STT_PROVIDER GLADIA_API_KEY OPENAI_API_KEY].each { |name| old.key?(name) ? ENV[name] = old[name] : ENV.delete(name) }
  end

  def test_recording_publication_escapes_provider_content_and_exposes_downloads
    Dir.mktmpdir do |root|
      id = 'a' * 40 + '-1000'
      raw = File.join(root, 'raw', id)
      FileUtils.mkdir_p(File.join(raw, 'audio'))
      File.write(File.join(raw, 'audio', 'test.webm'), 'synthetic test marker')
      File.write(File.join(raw, 'events.xml'), '<recording><metadata meetingId="summary-test" meetingName="Test &amp; recap"/><event timestamp="1000"/><event timestamp="3000"/></recording>')
      state = File.join(root, 'summary', id)
      FileUtils.mkdir_p(state)
      File.write(File.join(state, 'audio.ogg'), 'Already rendered test audio')
      props = { 'recording_dir' => root, 'published_dir' => File.join(root, 'published'), 'playback_protocol' => 'https', 'playback_host' => 'bbb.example.test' }
      provider = Provider.new
      BigBlueButton.logger = Logger.new(File::NULL)
      recording = BigBlueButton::Summary::Recording.new(id, props: props, provider: provider)
      recording.process
      recording.publish
      assert provider.calls == 1
      folder = File.join(root, 'published', 'summary', id)
      html = File.read(File.join(folder, 'index.html'))
      assert !html.include?('<script>')
      assert html.include?('&lt;script&gt;')
      assert html.include?('summary.txt')
      assert html.include?('transcript.txt')
      assert File.read(File.join(folder, 'transcript.txt')) == RESULT['transcription']['full_transcript']
      xml = Nokogiri::XML(File.read(File.join(folder, 'metadata.xml')))
      assert xml.at_xpath('/recording/playback/format').text == 'summary'
      assert xml.at_xpath('/recording/state').text == 'published'
      assert xml.at_xpath('/recording/playback/link').text == "https://bbb.example.test/summary/#{id}/index.html"
      assert File.exist?(File.join(root, 'status', 'published', "#{id}-summary.done"))
      assert File.stat(File.join(folder, 'summary.json')).mode & 0777 == 0644
    end
  end
end

test = SummaryTest.new
SummaryTest.instance_methods(false).grep(/^test_/).each do |name|
  test.public_send(name)
  puts "#{name}: PASS"
end
