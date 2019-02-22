FROM ubuntu:18.10

COPY ./votr /opt/votr
COPY ./static /static/

CMD /opt/votr
