# Dzień 4

## Console Manager

Niestety próby z ncurses zakończyły się fiaskiem.

Udało się co prawda zcompilowac rozszerzenie pod php 8.2, jednak wydaje się, że nie działa ono poprawnie.

Dodatkowo ncurses obsługuje ograniczoną paletę kolorów i nie wspiera bezpośrednio 8-bitowej palety kolorów ANSI.

Zamniast tego planuję napisac własną mini bibliotekę do obsługi konsoli, okienek

