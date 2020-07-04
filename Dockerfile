FROM centurylink/ca-certs
WORKDIR /app
COPY ./wfs-ls /app
COPY ./icons /app/icons
COPY ./migrations /app/migrations
COPY ./demodata/avatars /app/avatars

CMD ["/app/wfs-ls", "--reset"]