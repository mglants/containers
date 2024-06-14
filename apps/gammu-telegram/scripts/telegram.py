# Receive messages from GSM modem via Gammu to Telegram
# https://github.com/captainmarshin/phone/
#
#!/usr/bin/env python

import os
import sys
import requests
from urllib.parse import quote
numparts = int(os.environ["DECODED_PARTS"])
phone = os.environ["GAMMU_NUMBER"]
sender = str(os.environ["SMS_1_NUMBER"])
botkey = os.environ["TELEGRAM_BOTKEY"]
chatid = os.environ["TELEGRAM_CHATID"]
text = ""

if numparts == 0:
    text = os.environ["SMS_1_TEXT"]
else:
    for i in range(1, numparts + 1):
        varname = "DECODED_%d_TEXT" % i
        if varname in os.environ:
            text = text + os.environ[varname]
text = text.replace("<#>", "")
# Construct the Telegram message with HTML formatting
html_formatted_message = "<b>{}</b>\n{} <b>({}</b>)".format(phone, text, sender)
encoded_message = quote(html_formatted_message, safe='')
headers = { "Accept": "application/json", "Content-Type": "application/json" }
# Construct the Telegram API URL with the properly formatted message
chat = "https://api.telegram.org/bot{}/sendMessage?chat_id={}&text={}&parse_mode=HTML".format(botkey, chatid, encoded_message)
# Save the chat_url to a log file
log_filename = "/dev/stdout"
with open(log_filename, "a") as log_file:
    log_file.write(chat + "\n")
notify = requests.get(chat, headers = headers)