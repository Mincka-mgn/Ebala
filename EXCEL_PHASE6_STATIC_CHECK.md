# Ebala, этап 6: выполненные проверки (без компиляции)

```
PASS: fixed sample capacity independent of c.Intervals
PASS: sample endpoints use Excel formulas
PASS: legacy interval rescaling not used for primary case sheet
PASS: dynamic sample code does not read Go interval coordinates
PASS: multi-bay grid uses editable span and number of bays
PASS: step grid uses both editable roof widths
PASS: dome uses independently calculated projection
PASS: paired roof grid uses two editable widths
PASS: live chart gets independent sample count
PASS: live chart uses midpoint x, not old left endpoints
PASS: midpoint has own Excel formula
PASS: reduced snow is conditional on current January temperature
PASS: report note does not copy Go-computed μ
PASS: angle input has an editable mode selector
PASS: rise-derived angle calculated in Excel
PASS: cone polar angle refers to computed slope
PASS: cone polar coefficient uses computed slope
PASS: sphere-cone geometry uses coefficient angle and block suffix
PASS: prepared unit tests for variable interval count
PASS: 19 static checks. No Go compilation and no XLSX execution.
PASS: 3 rows (18 cells) of СП 20.13330.2016 table 10.2
PASS: 4 column arrays (44 values) of СП 20.13330.2016 table 11.2
PASS: 8 geometry/grid examples and 3 slope-from-rise examples
LIMIT: no Go compilation, no generated XLSX, no Microsoft Excel recalc
```
