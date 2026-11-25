# EDU cumulative seed — 02

Это самостоятельный проект для занятия 02. Состояние: **до lesson 02**.
Учебный срез: **health-only backend и пустые контракты**. Сервисы этого состояния: **db, backend (health-only)**.

## Перед запуском

Нужен Docker Desktop с работающим daemon. Все данные ниже синтетические;
файл `.env` создаётся локально и не должен попадать в Git или архив.
Проверяйте команды по очереди: при ненулевом коде остановитесь и
исправьте указанную причину до следующей команды.

```bash
docker info || exit $?
cp .env.example .env || exit $?
docker compose --env-file .env config --quiet || exit $?
docker compose --env-file .env build || exit $?
docker compose --env-file .env up -d --wait --wait-timeout 120 || { status=$?; docker compose --env-file .env down --remove-orphans; exit "$status"; }
docker compose --env-file .env ps
```
Если `config` не проходит, проверьте `.env` и обязательные значения. Если
`build` не проходит, оставьте полный вывод Docker для диагностики. После
`up` команда `ps` должна показать ожидаемые сервисы в рабочем состоянии.

## Подготовка двух учебных пользователей

После запуска сервисов выполните команду из корня проекта. Она дважды попросит
ввести пароль в скрытом интерактивном вводе — отдельно для A и B; значения не
печатаются и не записываются в Git:

```bash
docker compose --env-file .env run --rm backend seed
```

Адрес backend для локального HTTP-клиента: `http://127.0.0.1:8080`. Используйте
`learner-a@example.invalid` и `learner-b@example.invalid` в разных cookie jars.

## Проверка состояния

```bash
check_status=0
docker compose --env-file .env exec -T backend /usr/local/bin/probe || check_status=$?
docker compose --env-file .env ps
docker compose --env-file .env down --remove-orphans
if [ "$check_status" -ne 0 ]; then printf 'Проверка завершилась с кодом %s\n' "$check_status"; exit "$check_status"; fi
```
`exec` обращается к уже запущенному контейнеру backend. Команда `down`
выполняется после сохранения кода проверки и не превращает отказ probe в успех.

Личные резюме, пароли, токены, `.env`, node_modules и отчёты не являются
материалами проекта. В этом состоянии проверяются только DB и backend API;
внешних сервисов и frontend здесь нет.
