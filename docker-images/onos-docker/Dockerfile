FROM onosproject/onos:2.7-latest

WORKDIR /root/onos

COPY ./run.sh /root/onos/run.sh
RUN chmod +x /root/onos/run.sh

ENTRYPOINT ["/bin/sh"]
CMD ["-c", "/root/onos/run.sh"]