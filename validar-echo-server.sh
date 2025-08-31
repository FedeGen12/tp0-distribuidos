#!/bin/sh

HOST=$(grep "SERVER_IP" server/config.ini | cut -d'=' -f2 | xargs)
PORT=$(grep "SERVER_PORT" server/config.ini | cut -d'=' -f2 | xargs)
MSG="Esto es Boca"

RESPONSE=$(docker run --rm --network=tp0_testing_net --entrypoint sh subfuzion/netcat -c "echo \"$MSG\" | nc -w 20 $HOST $PORT")

if [ "$RESPONSE" = "$MSG" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi

