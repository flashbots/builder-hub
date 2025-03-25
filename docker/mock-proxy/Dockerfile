# BuilderHub expects measurements headers. For testing purposes, we just mock them with
# this nginx-based proxy container.
FROM nginx:1.27
COPY ./docker/mock-proxy/nginx-default.conf /etc/nginx/conf.d/default.conf

