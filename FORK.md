# Зачем этот форк

`github.com/tatnet-ru/libovsdb` — форк
[`ovn-kubernetes/libovsdb`](https://github.com/ovn-kubernetes/libovsdb) от тега
**v0.8.1** с исправлениями гонок при реконнекте, которые в апстриме лежат
непринятыми.

Форк, а не `replace`: путь модуля переписан на `github.com/tatnet-ru/libovsdb`,
поэтому зависимость видна прямо в `go.mod` потребителя, а не прячется в
директиве, о которой легко забыть.

## Что здесь есть сверх v0.8.1

Взято из [PR #460](https://github.com/ovn-kubernetes/libovsdb/pull/460)
(«Miscellaneous data race fixes detected by test runs with -race option»,
автор [@booxter](https://github.com/booxter)):

* **`cache`: read-локи в `Mapper()` и `DatabaseModel()`.** Оба читали
  `t.dbModel` без блокировки, а `Purge()` пишет его под мьютексом при
  переподключении. Гонка наблюдалась в проде: клиент, переживший рестарт
  соседнего сервера OVSDB, оставался с застывшим кешем и переставал писать.
* **`client`: не закрывать `trafficSeen` при дисконнекте**, а помечать клиент
  флагом `disconnected` — закрытие канала гонялось с отправкой из `transact()`
  и роняло процесс паникой «send on closed channel».
* **`client`: запускать `handleDisconnectNotification` после всех `Add()`** у
  `handlerShutdown`, иначе `Wait()` при переподключении мог уехать раньше
  регистрации обработчиков.

Плюс регрессионный тест `TestTransactDuringDisconnectNoPanic` из того же PR.

## Почему не из апстрима

На момент форка (31.08.2026):

* последний релиз апстрима — **v0.8.1 от 05.08.2025**, то есть год назад, при
  живом `main` (78 коммитов вперёд). Релизного канала фактически нет;
* PR #460 висит с 06.02.2026 в состоянии draft, конфликтует с `main`, **ноль
  ревью и ноль комментариев**;
* в `main` этих исправлений тоже нет — `Mapper()` и `DatabaseModel()` там
  по-прежнему без блокировок.

Ждать было нечего, а гонку мы наблюдали на живом кластере.

## Известное ограничение

`TestTransactDuringDisconnectNoPanic` пропускается под `-race`: там срабатывает
гонка внутри `github.com/cenkalti/rpc2` (ответ на серверный запрос пишется в тот
же `json.Encoder` и тот же `net.Conn`, куда параллельно пишет наш запрос). Есть
и в v1.0.4, и в v1.0.5 — чинить надо у rpc2. Без `-race` тест выполняется и
проверяет главное: отсутствие паники при транзакции во время дисконнекта.

## Обновление с апстрима

```bash
git fetch upstream
git rebase upstream/main            # или на нужный тег
grep -rl 'ovn-kubernetes/libovsdb' --include='*.go' --include=go.mod . \
  | xargs sed -i '' 's|github.com/ovn-kubernetes/libovsdb|github.com/tatnet-ru/libovsdb|g'
go build ./client/... ./cache/... && go test ./cache/... ./client/...
```

`example/` в апстриме не собирается без `go generate` (нужны модели
`vswitchd`) — это состояние самого репозитория, а не следствие форка.
