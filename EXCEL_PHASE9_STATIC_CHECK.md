# Ebala — этап 9: статическая проверка

Пройдено: **45/45**.

Проверяются исходники, состав резерва и независимые арифметические примеры. Это НЕ исполнение Go, НЕ генерация XLSX и НЕ проверка Excel.

| Проверка | Результат |
|---|---|
| Previous source checkpoint preserved in this directory | ОК |
| PROJECT.md is identical to stage 8 | ОК |
| Explicit 12 bay source limit | ОК |
| Reserve individual left and right bay-side alternatives | ОК |
| Reserve both two-adjacent-slope arrangements | ОК |
| Reserve all four alternating one-slope arrangements | ОК |
| Reserve 11 adjacent multi-vault alternatives | ОК |
| Recalculate active bay condition from Excel B17 | ОК |
| Live count >= 2 and <= 12 | ОК |
| Individual bay activation conditioned on edited count | ОК |
| Per-bay live coordinate formulas | ОК |
| Empty inactive-bay coordinates | ОК |
| Prevent mu formula evaluating on empty grid rows | ОК |
| Reserve six spatial round sectors without Go Polar dependency | ОК |
| One shared spatial sheet rather than six duplicated grids | ОК |
| Shared sector spatial grid has independent inputs | ОК |
| Shared sector grid contains all six independent mu columns | ОК |
| Shared sector grid has independent normative load columns | ОК |
| Shared sector grid has independent design load columns | ОК |
| No source Go Polar or Go mu copied into shared sector grid | ОК |
| Angular grid one degree 0..360 | ОК |
| Radial grid 12 intervals | ОК |
| Sector active condition reevaluated from geometry | ОК |
| Sector sheet references renamed for combined output | ОК |
| Original sector Polar sheets no longer duplicated | ОК |
| Sheet order recognizes sector sheet | ОК |
| Sector grid columns A to V are readable | ОК |
| New unit tests prepared but not executed | ОК |
| Independent bay geometry n=2: continuous and covers nL | ОК |
| Independent bay geometry n=3: continuous and covers nL | ОК |
| Independent bay geometry n=8: continuous and covers nL | ОК |
| Independent bay geometry n=12: continuous and covers nL | ОК |
| Independent angular sectors at beta=0°: one half + one quarter | ОК |
| Independent angular sectors at beta=45°: one half + one quarter | ОК |
| Independent angular sectors at beta=89°: one half + one quarter | ОК |
| Independent angular sectors at beta=90°: one half + one quarter | ОК |
| Independent angular sectors at beta=135°: one half + one quarter | ОК |
| Independent angular sectors at beta=179°: one half + one quarter | ОК |
| Independent angular sectors at beta=180°: one half + one quarter | ОК |
| Independent angular sectors at beta=225°: one half + one quarter | ОК |
| Independent angular sectors at beta=269°: one half + one quarter | ОК |
| Independent angular sectors at beta=270°: one half + one quarter | ОК |
| Independent angular sectors at beta=315°: one half + one quarter | ОК |
| Independent angular sectors at beta=359°: one half + one quarter | ОК |
| Independent angular sectors at beta=360°: one half + one quarter | ОК |
