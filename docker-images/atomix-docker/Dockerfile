FROM atomix/atomix:3.1.12

WORKDIR /

COPY ./agent /root/agent
RUN chmod +x /root/agent

ENTRYPOINT ["/root/agent"]