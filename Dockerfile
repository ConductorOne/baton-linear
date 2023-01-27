FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-linear"]
COPY baton-linear /