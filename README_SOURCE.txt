Ebala 0.53 — исходные тексты программы
Go, ресурсы, климатические справочники, регрессионные тесты.
Проверка: GO111MODULE=off go test ./...
Сборка: build_windows.bat (или GOOS=windows GOARCH=amd64 GO111MODULE=off go build -ldflags="-H=windowsgui -s -w" -o Ebala_0.53.exe .).
Ограничения расчёта приведены в CHANGES_0.41.txt.
Для регрессионной проверки: GO111MODULE=off go test ./... -count=1.
Для контрольного экспорта: EBALA_QA_OUT=<папка> GO111MODULE=off go test -run Test041MultipleSnowAndIceExports -v.

Для сборки версии Ebala 0.53 используйте исходники из данного архива. Перечень изменений: CHANGES_0.53.txt.

Изменения версии 0.46: CHANGES_0.46.txt.

Важно: используйте -ldflags="-H=windowsgui": иначе Windows создаёт консоль и её закрытие завершает Ebala.
Go 1.23.x не поддерживает Windows 7; для Windows 7 необходима отдельная сборка с Go 1.20.x и её проверка на целевой ОС.

Рисунок сферы с конусом snow_B15a_approved_geometry.png — утверждённый пользователем PNG без изменения байтов.
Формирование: GO111MODULE=off go test ./... -count=1; сборка через build_windows.bat.

Ограничения проверки выпуска 0.53: см. EXCEL_PHASE12_QA.md. Этот выпуск является проверочным; все критерии Excel ещё не закрыты.
