FROM hashicorp/vault:1.21
USER root
RUN apk add --no-cache python3 py3-pip openssl
RUN pip3 install --no-cache-dir --break-system-packages PyJWT cryptography
COPY docker/vault-init/init.sh /init.sh
RUN chmod +x /init.sh
ENTRYPOINT ["/bin/sh", "/init.sh"]
