# !/bin/bash
service selfweb3 stop

# build
./build.sh

# deploy
rm ./deploy/selfweb3
rm -rf ./deploy/rsweb

cp ./selfweb3 ./deploy/
cp -r ./rsweb ./deploy/

service selfweb3 start