FROM scratch

COPY staticreg /

ENTRYPOINT ["/staticreg"]
