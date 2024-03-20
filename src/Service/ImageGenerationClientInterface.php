<?php

namespace App\Service;

interface ImageGenerationClientInterface
{
    public function call(string $prompt): string;
}
