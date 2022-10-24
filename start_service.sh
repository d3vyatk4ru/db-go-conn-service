echo "Start docker container"

docker run -p 3306:3306 -v /home/d3vyatk4ru/Desktop/lectures-2022-2/06_databases/99_hw/db/_sql:/docker-entrypoint-initdb.d \
-e MYSQL_ROOT_PASSWORD=1234 \
-e MYSQL_DATABASE=golang \
-d mysql

docker exec -it mysql-server-db "bash"

mysql -u root -ppassword 1234 < docker-entrypoint-initdb.d/sample_db.sql 