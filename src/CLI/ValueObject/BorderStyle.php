<?php

class BorderStyle
{
    public function __construct(
        private int $color,
        private string $topLeftcharacter = '+',
        private string $topCharacter = '-',
        private string $topRightcharacter = '+',
        private string $leftcharacter = '|',
        private string $rightcharacter = '|',
        private string $bottomLeftcharacter = '+',
        private string $bottomCharacter = '-',
        private string $bottomRightcharacter = '+',
    ) {
        // TODO: validate character have length = 1
        // TODO: validate color
    }
}
