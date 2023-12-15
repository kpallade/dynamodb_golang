1. Créer les dossiers docker/dynamodb pour heberger les datas
```
$ mkdir -p docker/dynamodb
```

2. Changer les permissions
```
$ chown -R 1000:1000 docker
```

3. Lancer le container
```
$ docker-compose up -d
```

4. Créer profil aws
```
$ aws configure --profile localhost
AWS Access Key ID [None]: 
AWS Secret Access Key [None]: 
Default region name [None]: localhost
Default output format [None]: 
```

5. Executer le code
```
$ go mod tidy
$ go run main.go
```