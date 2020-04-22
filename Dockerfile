FROM centurylink/ca-certs
WORKDIR /app
COPY ./wfs-ls /app
COPY ./icons /app/icons
COPY ./migrations /app/migrations

CMD ["/app/wfs-ls"]