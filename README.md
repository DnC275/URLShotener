# URLShotener
Веб-сервис на Go для укорачивания ссылок.
## Использование
* ### Создание ссылки:
    Чтобы создать ссылку необходимо послать POST-запрос на сервер по url http://*host*:*port*/create/.
В теле запроса передать json с параметром longUrl, в значении которого указать url, который вы хотите сократить.
    #### Пример:
    Сервер запущен на localhost на 8000 порту. 
Тогда адрес запроса будет http://localhost:8000/create/. 

  body запроса:
  ```json
  {
    "longUrl": "https://www.google.com/"
  }
  ```
  Ответ:
  ```json
  {
    "shortUrl": "http://127.0.0.1:8000/DnAqqr2err"
  }
  ```
  Здесь, DnAqqr2err (т.е. 10 символов, которые идут после адреса сервера) - уникальный идентификатор полученной ссылки. 
В дальнейшем - shortUrlPK.
* ### Получение оригинального url:
    GET-Запрос на http://*host*:*port*/expand?shortUrlPK=*shortUrlPK*
    #### Пример:
    Запрос по url http://localhost:8000/expand?shortUrlPK=DnAqqr2err

    Ответ: 
  ```json
  {
    "longUrl": "https://www.google.com/"
  }
  ```
* ### Редирект
    Чтобы попасть на оригинальный url просто обратитесь на полученный shortUrl.

## Локальный запуск сервиса
* ### Из исходников:
    
  * ```shell
    git clone https://github.com/DnC275/URLShotener/tree/master
  * ```shell
    cd URLShotener
  * ```shell
    go get .
  При старте сервера можно задать параметр -storage, который указывает какой тип хранилища будет использовать сервис: in-memory или базу данных postgres.
  По умолчанию -storage=in-memory.
  
    Если вы хотите использовать PostgreSQL, то необходимо передать -storage=postgres, после чего задать параметры **db_host**(хост базы данных), **db_port**(порт бд), **db_user**(юзера для доступа к бд), **db_pwd**(пароль от юзера) и **db_name**(имя существующей бд).
Если не задать эти параметры, то будут использовны значения по умолчанию: **db_host**=127.0.0.1, **db_port**=5432, **db_user**=postgres, **db_pwd**=postgres, **db_name**=urlshortener.
Значения по умолчания можно поменять, изменив значения в файле .env
    #### Примеры:
    для использования in-memory хранилища:
    * ```shell
      go run main.go
    для использования PostgreSQL:
    * ```shell
      go run main.go -storage=postgres -db_host=127.0.0.1 -db_port=5050 -db_user=user -db_pwd=pass -db_name=mydatabase 
  
* ### Через docker-compose:
    * ```shell
      git clone https://github.com/DnC275/URLShotener/tree/master
    * ```shell
      docker build -t myapp .
    * ```shell
      docker run -p 8000:8000 myapp [arg1=..., arg2=...,]
      ```
    Для развертывания PostgresSQL можно воспользоваться моим docker-compose файлом:
    ```shell
  docker-compose up
  ```
    В этом случае, при запуске сервиса через docker, необходимо указать сеть и поменять хост базы данных на postgres:
  ```shell
  docker run -p 8000:8000 --net urlshortener myapp -storage postgres -db_host=postgres
  ```
  Также вы можете использовать докер образ с моего dockerhub: dnc275/urlshortener:latest
      
    
  