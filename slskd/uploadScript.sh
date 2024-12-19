#!/bin/bash

DOCUMENT=$2

echo "db.${COLLECTION_NAME}.insert(${DOCUMENT})" | mongo --username $MONGO_USER --password $MONGO_PASS --authenticationDatabase admin http://web-graffiti-gluetun:$MONGO_PORT/$MONGO_DB
