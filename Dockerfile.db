FROM postgres:15
ENV POSTGRES_DB postgres
ENV POSTGRES_HOST_AUTH_METHOD=trust
COPY ./schema/*.sql /docker-entrypoint-initdb.d/