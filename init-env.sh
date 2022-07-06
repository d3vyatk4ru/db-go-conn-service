sudo apt-get update -y\
&& sudo apt-get upgrade -y\
&& go mod init main.go -y\
&& go get github.com/microsoft/go-mssqldb -y\
&& go get github.com/gorilla/mux -y\
&& docker start && docker-compose up -d\
&& docker exec -it sql-server-db "bash"\
&& for query in /var/lib/mssqlql/data/*.sql\
do /opt/mssql-tools/bin/sqlcmd -U sa -P super_PWD_go -l 10 -e -i $$query done
