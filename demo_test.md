======
postgres compose
------
services:
  postgres:
    container_name: postgres
    image: postgres:17
    environment:
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PW}
      - POSTGRES_DB=${POSTGRES_DB} #optional (specify default database instead of $POSTGRES_DB)
    ports:
      - "5432:5432"
    restart: always

  pgadmin:
    container_name: pgadmin
    image: dpage/pgadmin4:latest
    environment:
      - PGADMIN_DEFAULT_EMAIL=${PGADMIN_MAIL}
      - PGADMIN_DEFAULT_PASSWORD=${PGADMIN_PW}
    ports:
      - "5050:80"
    restart: always

======
kafka compose
------
services:
  kafka:
    image: apache/kafka:latest
    container_name: kafka
    restart: unless-stopped
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@localhost:9093
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: 0
      KAFKA_NUM_PARTITIONS: 3
    ports:
      - 9092:9092

======
hadoop compose
------
services:
   namenode:
      platform: linux/amd64
      container_name: namenode
      image: apache/hadoop:3
      hostname: namenode
      restart: unless-stopped
      command: ["hdfs", "namenode"]
      ports:
        - 9870:9870
        - 8020:8020
      env_file:
        - ./hadoop_config
      environment:
          ENSURE_NAMENODE_DIR: "/tmp/hadoop-root/dfs/name"
      networks:
        - hadoop_network
   datanode1:
      platform: linux/amd64   
      container_name: datanode1
      depends_on:
        - namenode
      image: apache/hadoop:3
      command: ["hdfs", "datanode"]
      restart: unless-stopped
      env_file:
        - ./hadoop_config 
      networks:
        - hadoop_network
   datanode2:
      platform: linux/amd64   
      container_name: datanode2   
      depends_on:
        - namenode
      restart: unless-stopped
      image: apache/hadoop:3
      command: ["hdfs", "datanode"]
      env_file:
        - ./hadoop_config      
      networks:
        - hadoop_network
   datanode3:
      platform: linux/amd64   
      container_name: datanode3 
      depends_on:
        - namenode
      restart: unless-stopped
      image: apache/hadoop:3
      command: ["hdfs", "datanode"]
      env_file:
        - ./hadoop_config   
      networks:
        - hadoop_network
   resourcemanager:
      platform: linux/amd64   
      container_name: resourcemanager      
      image: apache/hadoop:3
      hostname: resourcemanager
      command: ["yarn", "resourcemanager"]
      restart: unless-stopped
      ports:
         - 8088:8088
      env_file:
        - ./hadoop_config
      volumes:
        - ./test.sh:/opt/test.sh
      networks:
        - hadoop_network
   nodemanager:
      platform: linux/amd64   
      container_name: nodemanager
      image: apache/hadoop:3
      restart: unless-stopped
      command: ["yarn", "nodemanager"]
      env_file:
        - ./hadoop_config
      networks:
        - hadoop_network

networks:
  hadoop_network:
    name: hadoop_network


======
hadoop_config
------
CORE-SITE.XML_fs.default.name=hdfs://namenode
CORE-SITE.XML_fs.defaultFS=hdfs://namenode
HDFS-SITE.XML_dfs.namenode.rpc-address=namenode:8020
HDFS-SITE.XML_dfs.replication=1
HDFS-SITE.XML_dfs.permissions.enabled=false
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.maximum-applications=10000
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.maximum-am-resource-percent=0.1
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.resource-calculator=org.apache.hadoop.yarn.util.resource.DefaultResourceCalculator
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.root.queues=default
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.root.default.capacity=100
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.root.default.user-limit-factor=1
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.root.default.maximum-capacity=100
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.root.default.state=RUNNING
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.root.default.acl_submit_applications=*
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.root.default.acl_administer_queue=*
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.node-locality-delay=40
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.queue-mappings=
CAPACITY-SCHEDULER.XML_yarn.scheduler.capacity.queue-mappings-override.enable=false


======
kafka exporter for prometheus
------
services:
  kafka-exporter:
    image: danielqsj/kafka-exporter 
    command: ["--kafka.server=kafka:9092"]
    ports:
      - 9308:9308
    restart: unless-stopped

networks:
  default:
    name: kafka_default
    external: true

======
mosquitto
------
services:
  mosquitto:
    image: eclipse-mosquitto:latest
    container_name: mosquitto
    restart: unless-stopped
    ports:
      - "1883:1883"
      - "9001:9001" # For WebSockets, if needed
    volumes:
      - ./config:/mosquitto/config
      - ./data:/mosquitto/data
      - ./log:/mosquitto/log
    command: mosquitto -c /mosquitto/config/mosquitto.conf

======
mosquitto config
------
persistence true
persistence_location /mosquitto/data/
log_dest file /mosquitto/log/mosquitto.log
allow_anonymous true
listener 1883
------





======
support utils

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


