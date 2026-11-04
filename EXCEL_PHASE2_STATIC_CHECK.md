# Статическая проверка исходников после этапа 2

Проверено 15 признаков (прошло 15). Это **не компиляция, не выполнение Go-кода и не проверка XLSX в Excel**.

| Признак | Итог |
|---|---|
| gololёd height grid 320 | Найден |
| ice grid links to input B12,B18,B19 | Найден |
| ice calc heights reference grid sheet | Найден |
| ice calc uses live table 12.3 | Найден |
| ice calc uses live table 12.4 | Найден |
| wind in generic summary always linked | Найден |
| round polar computes beta from input | Найден |
| round polar computes rise from input | Найден |
| round polar no numerical point export | Найден |
| old polar Go exporter removed | Найден |
| old Go numerical audit helper removed | Найден |
| GO tests created for next phase | Найден |
| project unchanged from phase1 | Найден |
| no new binaries | Найден |
| none of generated XLSX | Найден |

Не проводились: go test, go build, экспорт новой XLSX, расчёт формул в Microsoft Excel, проверка Windows.
