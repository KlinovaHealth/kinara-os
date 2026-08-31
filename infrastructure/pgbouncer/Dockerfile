FROM alpine:3.20
RUN apk add --no-cache pgbouncer \
    && mkdir -p /etc/pgbouncer \
    && chown pgbouncer:pgbouncer /etc/pgbouncer
USER pgbouncer
ENTRYPOINT ["/usr/bin/pgbouncer"]
CMD ["/etc/pgbouncer/pgbouncer.ini"]
