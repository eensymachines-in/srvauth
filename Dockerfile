FROM kneerunjun/raspbian_jessi_go15_3:latest
# Above image shall create the go environment and the necessary folders

RUN mkdir -p /var/local/srvauth
RUN touch /var/local/srvauth/device.log

RUN mkdir -p /home/pi/eensymachines-in/srvauth
WORKDIR /home/pi/eensymachines-in/srvauth

# now creating folder for the code and repository 
# since we commit to use 1.15 and above we would be using GO Modules
COPY . .
RUN go mod download 
RUN go build -o srvauth .