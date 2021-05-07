
Please note from the env dev file, only `LOGDIR` and `SOCKSDIR` need corresponding volume map on the host machine
rest of the directories dont need to be `mapped` on the host machine.

```
OWNER=kneerunjun@gmail.com
FLOG=false
VERBOSE=true
TTYSTDIN=true
LOGDIR=/var/log/eensymachines
SOCKSDIR=/run/eensymachines
REPODIR=/usr/src/eensymachines
GOBINDIR=/home/pi/go/bin
BINDIR=/usr/bin
TIMEZ=/etc/timezone
LCLTM=/etc/localtime
RELAYS=IN1,IN2,IN3,IN4
RELAYPINS=40,38,36,32
RELAYINVT=true
GPIOMEM=/dev/gpiomem
I2C=/dev/i2c-1
AUTHBASEURL=http://auth.eensymachines.in
REGBASEURL=http://lumin.eensymachines.in/api/v1/devices
```

Setting up the host machine 

```sh
groupadd eensymachines
usermod -aG eensymachines pi
mkdir -p /run/eensymachines 
chown -R :eensymachines /run/eensymachines
chmod -R g+w /run/eensymachines
mkdir -p /var/log/eensymachines
chown -R :eensymachines /var/log/eensymachines
chmod -R g+w /var/log/eensymachines
```
