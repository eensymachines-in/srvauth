FROM kneerunjun/raspbian_jessi_go15_3:latest
# All what we do here is create a suitable src directory on the guest machine
# send the sum/mod files > download the go modules 
# then copy the src code from host to guest 
# build executable srvauth and install in appropriate location 
ARG SRC
ARG BIN

RUN mkdir -p ${SRC}
WORKDIR ${SRC}

COPY go.sum go.mod  ./
RUN go mod download 
# downloading the module on a distinct layer
COPY . .
RUN go build -o ${BIN}/srvauth .
