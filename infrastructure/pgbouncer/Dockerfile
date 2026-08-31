FROM alpine:3.20
RUN apk add --no-cache pgbouncer \
    && addgroup -S -g 70 pgbouncer 2>/dev/null || true \
    && adduser -S -u 70 -G pgbouncer pgbouncer 2>/dev/null || true
USER 70
ENTRYPOINT ["/usr/bin/pgbouncer"]
CMD ["/etc/pgbouncer/pgbouncer.ini"]
