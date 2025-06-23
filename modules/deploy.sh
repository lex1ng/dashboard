

docker build --platform=linux/amd64  -t iregistry.baidu-int.com/cloudbed-system/kubernetesui/dashboard-api:"$1" -f ./api/Dockerfile-bak ./api
#docker build --platform=linux/amd64  -t iregistry.baidu-int.com/cloudbed-system/kubernetesui/dashboard-api:"$1"  ./api

#docker push iregistry.baidu-int.com/cloudbed-system/kubernetesui/dashboard-api:"$1"
