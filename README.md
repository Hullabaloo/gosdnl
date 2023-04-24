# Комментарии к тестовому заданию

## Содержание

- [Комментарий к реализации](#cmnt)
- [О задании и определение области необходимых данных](#s0)
- [A: Метод: Импорт/Обновление данных](#s11)
- [A: Метод: Получение текущего состояния данных](#s12)
- [A: Метод: Получение списка имен](#s13)
- [B: Установка и запуск](#install)
- [C: Описание алгоритма для более эффективного обновления данных](#algupdate)

## Комментарий к реализации <a name = "cmnt"></a>

В коде задания хардкодом указаны параметры доступа к БД, для рабочего приложения эти данные лучше хранить в переменных в .env файле.

При вызове методов и возникновении непредвиденных состояний будет правильным возвращать в ответе информацию о возникшей проблеме.

В методе get_names - было бы правильным возвращать соответствующий ответ об ошибке, если параметр "name" не был указан.

## О задании и определение области необходимых данных <a name = "s0"></a>

Текст задания подразумевает реализацию следующих методов:

- метод для загрузки данных из внешнего источника (предоставлена ссылка на XML документ), выборки необходимых данных и сохранение/обновление данных в локальной БД
- метод для контроля статуса приложения (данных нет, обновление в процессе, данные готовы к использованию)
- метод для вывода списка всех возможных имен человека по заданным параметрам

Т.к. в задании основной задачей является загрузка данных из внешнего источника и дальнейшее использование данных в локальной БД для поиска по имени/фамилии, то областью необходимых данных были определены следующие поля: 
```
uid, firstname, lastname
```
Другие поля данных во внешнем источнике не представляют ценности в рамках этого задания.

## Импорт/обновление данных <a name = "s11"></a>

Для запуска процедуры импорта или обновления данных необходимо выполнить запрос по следующей ссылке:
```
localhost:8080/update
```

## Получение текущего состояния данных <a name = "s12"></a>

Для запуска процедуры проверки текущего сосотояния данных необходимо выполнить запрос по следующей ссылке:
```
localhost:8080/state
```

## Получение списка имен <a name = "s13"></a>

Для запуска процедуры получения необходимо выполнить запрос по следующей ссылке, включая необходимые GET параметры:
```
localhost:8080/get_names?name={SOME_VALUE}&type={strong|superstrong|weak}&option={full}
```
- в случае отсутствия в запросе параметра type, будет произведена выборка по типу "weak", т.к. он в любом случае включает в себя все возможные варианты других типов (stron, superstrong)
- был введен дополнительный тип "superstrong", который позволяет выбрать записи, имеющие точное совпадение по имени и фамилии, например:
``
localhost:8080/get_names?name=Subhi TUFAYLI&type=superstrong
``
- для типа "weak" был введен дополнительный параметр - "option=full", который позволяет выбрать записи, имеющие полное совпадение одного целого слова в имени или фамилии. В данном случае в SQL запросе используется регулярное выражение для создания массива слов, состоящего из отдельных слов, содержащихся в имени и фамилии (см. main.go:140). Пример запроса:
``
localhost:8080/get_names?name=Subhi&type=weak&option=full
``
- при использовании типа "weak" без параметра "option=full", запрос вернет все записи, в которых любая часть имени или фамилии содержит в себе запрашиваемый параметр "name". Пример запроса:
``
localhost:8080/get_names?name=bhi&type=weak
``

## Установка и запуск <a name = "install"></a>

docker-compose.yml содержит инструкции для разворачивания приложения на порту 8080

Для запуска приложения, необходимо в консоли перейти в папку с файлами приложения и запустить следующую команду:
```
docker-compose up
```
Дождаться компиляции необходимых пакетов, после чего можно выполнять запросы к соответсвующим методам методам приложения.

## Описание алгоритма для более эффективного обновления данных <a name = "algupdate"></a>

Описать алгоритм для более эффективного обновления данных при повторном вызове метода "localhost:8080/update".

В нашем случае (в нашей области данных для полей firstname, lastname):
- загрузить данные из внешнего источника
- в цикле проверить, если текущий UID в локальной БД, если нет - добавить запись
- если текущий UID уже существует в локальной БД, проверить изменения в полях firstname или lastname, если изменения есть - обновить запись в БД
- далее выбрать все UID из локальной БД и в цикле проверить, существует ли UID из локальной БД в загруженных данных из внешнего источника, если UID не найдет, то удалить запись из локальной БД.


### Дополнение А
В случае более обширной области данных (при наличии дополнительных полей данных - напр. данные документов, описание запрета и т.п.), выгоднее всего будет хранить в локальной БД в дополнительном поле хеш блока данных. При итерации проверять, изменился ли хеш блока данных из внешнего источника, и если изменился - обновить данные записи.

### Дополнение Б
При наличии в данных внешнего источника поля updated_at, процесс обновления данных упрощается, т.к. появляется возможность обновлять данные в локальной БД, в которых временная метка в поле updated_at меньше(старее), чем анологичная метка в поле данных внешнего источника.