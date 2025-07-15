-----------------
kafka :


cd ~/work/docker/kafka/kafka_2.13-4.0.0/bin
./kafka-topics.sh --bootstrap-server localhost:9092 --list
./kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic bz.hands.left


-----------------
rest :

./ut config_id_red.yaml > red.log 2>&1 &
./ut config_id_yellow.yaml > yellow.log 2>&1 &
./ut config.yaml > hands.log 2>&1 &


-----------------
coap : 


echo -n 'hello world 디바이스' | coap post coap://localhost/hands/left
echo -n '{ "msg" : "hello world 공장장"}' | coap post coap://localhost/hands/left
coap post coap://localhost:5683/hello -p '{"redfoot": "south"}'



----------------
mqtt :

mosquitto_pub -h localhost -t myTopic -m '{ "mqmsg": "서울에서 정선까지67" }'
mosquitto_pub -h localhost -t doll/sign -m '{ "mqmsg": "서울에서 정선까지67" }'

----------------
snmp :

sudo apt install snmpd
sudo systemctl enable snmpd
sudo systemctl start snmpd
sudo apt install snmp        # client

# version 3 user create.
sudo net-snmp-create-v3-user -ro -a SHA -A Str0ng@uth3ntic@ti0n -x AES -X Str0ngPriv@cy nagios

cf. https://library.nagios.com/documentation/step-by-step-guide-setting-up-snmp-on-ubuntu-24-for-nagios-xi/

# v2
snmpwalk -v2c -c public localhost
# v3
snmpwalk -v3 -u nagios -l authPriv -a SHA -A Str0ng@uth3ntic@ti0n -x AES -X Str0ngPriv@cy localhost

gosnmp 패키지 examples
oids := []string{"1.3.6.1.2.1.1.1.0", "1.3.6.1.2.1.1.3.0"}
첫번째는 os name,  두번째는 os booting 후 지난 시간.


