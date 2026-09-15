# Зачем этот форк

`github.com/tatnet-ru/libovsdb` — форк
[`ovn-kubernetes/libovsdb`](https://github.com/ovn-kubernetes/libovsdb) с
исправлениями гонок при реконнекте, которые в апстриме лежат непринятыми.

Форк, а не `replace`: путь модуля переписан на `github.com/tatnet-ru/libovsdb`,
поэтому зависимость видна прямо в `go.mod` потребителя, а не прячется в
директиве, о которой легко забыть.

## База

**Апстримный `main` @ `6acd868` (01.09.2026)** — это 84 коммита после
последнего релиза апстрима v0.8.1 (05.08.2025). Релизного канала у апстрима
фактически нет: за год после v0.8.1 ни одного тега, поэтому «догонять» можно
только по `main`.

Предыдущая база форка — тег v0.8.1 (наш тег v0.8.2, 31.08.2026).

## Что здесь есть сверх апстрима

Два хунка, оба из
[PR #460](https://github.com/ovn-kubernetes/libovsdb/pull/460)
(«Miscellaneous data race fixes detected by test runs with -race option»,
автор [@booxter](https://github.com/booxter)), в апстриме по-прежнему нет:

* **`cache`: read-локи в `Mapper()` и `DatabaseModel()`.** Оба читают
  `t.dbModel`, а `Purge()` пишет его под тем же мьютексом при переподключении.
  Гонка наблюдалась в проде: клиент, переживший рестарт соседнего сервера
  OVSDB, оставался с застывшим кешем и переставал писать.
* **`client`: запускать `handleDisconnectNotification` после всех `Add()`** у
  `handlerShutdown`, иначе `Wait()` при переподключении может уехать раньше
  регистрации обработчиков.

**Гонка живая, не теоретическая.** Замер 15.09.2026 на
`TestIntegration_ReconnectAfterRestart` из `services/ovn` под `-race`, против
чистого апстримного `main`:

```
WARNING: DATA RACE
Write at ... by goroutine 32:
  cache.(*TableCache).Purge()            cache/cache.go:1036
  client.(*ovsdbClient).monitor()        client/client.go:1053
  client.(*ovsdbClient).connect()        client/client.go:299
Previous read at ... by goroutine 144:
  client.api.Create()                    client/api.go:381
--- FAIL: TestIntegration_ReconnectAfterRestart (3.77s)
```

С хунками — `ok`, три прогона подряд.

## Что было в форке и больше не нужно

Третий хунк PR #460 — «не закрывать `trafficSeen` при дисконнекте» — апстрим
починил сам, по-своему и лучше: коммит
[`6d390f6`](https://github.com/ovn-kubernetes/libovsdb/commit/6d390f6)
обнуляет `o.trafficSeen` под `rpcMutex.Lock()`, а читается поле под
`rpcMutex.RLock()` в `transact()`; канал не закрывается вообще, поэтому
«send on closed channel» невозможен. Наш флаг `disconnected` с собственным
мьютексом снят, регрессионный тест `TestTransactDuringDisconnectNoPanic` и
файлы `client/race_detector_{on,off}.go` — тоже: апстрим принёс свой тест
`TestTrafficSeenNotClosedOnDisconnect`. Заодно ушло и известное ограничение
форка (наш тест пропускался под `-race` из-за гонки внутри `cenkalti/rpc2`).

В том же наборе из 84 коммитов апстрим закрыл ещё три гонки, которых у нас не
было: `6c8c8d4` (Echo), `1b1778d` (чтение флага shutdown в reconnect),
`66a1f0f` (проверка shutdown перед RPC).

## Обновление с апстрима

Порядок важен: **сначала путь модуля, потом хунки**. Если делать наоборот,
любой `git checkout upstream/main -- .` (например, чтобы что-то сверить)
перезапишет индекс и молча выбросит хунки из следующего коммита.

```bash
git fetch upstream
git checkout -b tatnet/main upstream/main

grep -rl 'ovn-kubernetes/libovsdb' --include='*.go' --include=go.mod --include=Makefile . \
  | xargs sed -i '' 's|github.com/ovn-kubernetes/libovsdb|github.com/tatnet-ru/libovsdb|g'
git commit -am 'Путь модуля → github.com/tatnet-ru/libovsdb'

# затем применить хунки из раздела выше и закоммитить отдельно
```

Проверять **тем, что попало в коммит**, а не тем, что лежит в рабочем дереве:

```bash
git show HEAD:cache/cache.go | sed -n '/func (t \*TableCache) Mapper/,+4p'   # должны быть RLock
go test -race ./cache/... ./client/...
cd ../services/ovn && go test -race -run TestIntegration_Reconnect .          # несущий замер
```

Документация (`README.md`, `CONTRIBUTING.md`, `MAINTAINERS`, `docs/`,
`.github/`) намеренно оставлена со ссылками на апстрим — это его документы, и
чем меньше расхождение, тем дешевле следующий rebase.

Ветка `main` в этом репозитории — чистое зеркало апстрима, без наших правок;
работа живёт в `tatnet/main`, версии — в наших тегах (`v0.8.2`, `v0.9.0`, …),
апстримные теги в этот репозиторий не заливаются.

`example/` в апстриме не собирается без `go generate` (нужны модели
`vswitchd`) — это состояние самого репозитория, а не следствие форка;
проверено на нетронутом `upstream/main`.
