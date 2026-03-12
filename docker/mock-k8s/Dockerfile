FROM python:3.12-alpine
RUN apk add --no-cache wget
COPY docker/mock-k8s/server.py /server.py
CMD ["python3", "/server.py"]
