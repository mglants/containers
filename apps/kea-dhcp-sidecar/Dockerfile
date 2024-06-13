FROM bitnami/kubectl:1.30.2 as kubectl
FROM docker.io/alpine:3.20.0

COPY --from=kubectl /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/

RUN apk update && \
    apk add --no-cache jq && \
    apk add --no-cache curl

WORKDIR /app

# Copy the generate_config() function to the container
COPY generate_config.sh /app/

# Make the generate_config() function executable
RUN chmod +x /app/generate_config.sh

# Run the generate_config() function on container start
CMD ["/app/generate_config.sh"]
