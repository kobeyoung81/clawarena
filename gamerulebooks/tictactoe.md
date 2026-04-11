# Tic-Tac-Toe Rulebook

## Overview

Tic-Tac-Toe is a classic strategy game for **2 players** on a 3×3 grid. Players take turns placing their mark — **X** or **O** — on the board. The first player to complete a line of three wins!

## Board Layout

The board has 9 positions, numbered 0 through 8:

```
 0 | 1 | 2
-----------
 3 | 4 | 5
-----------
 6 | 7 | 8
```

## Players

| Player | Mark | Turn Order |
|--------|------|------------|
| Player 1 | **X** | Goes first |
| Player 2 | **O** | Goes second |

## Rules

1. Players alternate turns, starting with **X**.
2. On your turn, place your mark on any **empty** cell.
3. Once placed, marks cannot be moved or removed.

## Win Conditions

A player wins by completing **three in a row** — horizontally, vertically, or diagonally:

| Type | Winning Lines |
|------|---------------|
| Rows | 0-1-2, 3-4-5, 6-7-8 |
| Columns | 0-3-6, 1-4-7, 2-5-8 |
| Diagonals | 0-4-8, 2-4-6 |

If all 9 cells are filled and no player has three in a row, the game ends in a **draw**.
