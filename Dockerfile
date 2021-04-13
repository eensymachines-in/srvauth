FROM kneerunjun/raspbian_jessi_go15_3:latest as raspjessgo
# Above image shall create the go environment and the necessary folders

RUN mkdir -p /home/pi/eensymachines-in/srvauth
WORKDIR /home/pi/eensymachines-in/srvauth

# now creating folder for the code and repository 
# since we commit to use 1.15 and above we would be using GO Modules
COPY go.sum go.mod  ./
RUN go mod download 


FROM kneerunjun/raspbian_jessi_go15_3:latest  

WORKDIR /home/pi/eensymachines-in/srvauth
COPY --from=raspjessgo $GOPATH/pkg/mod $GOPATH/pkg/mod
COPY . . 
RUN go build -o srvauth .