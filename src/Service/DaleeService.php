<?php

namespace App\Service;

use Symfony\Contracts\HttpClient\HttpClientInterface;
use DateTime;

class DaleeService implements ImageGenerationClientInterface
{
    public function __construct(
        private HttpClientInterface $client,
        private string $openAiToken,
    )
    {
    }

    public function call(string $prompt): string
    {
        echo $this->openAiToken;
        $response = $this->client->request('POST', 'https://api.openai.com/v1/images/generations', [
            'headers' => [
                'Authorization' => 'Bearer '.$this->openAiToken,
                'Content-Type' => 'application/json',
            ],
            'json' => [
                'model'  => 'dall-e-3',
                'prompt' => $prompt,
                'n'      => 1,
                'size'   => '1024x1024',
            ],
        ]);

        $content      = $response->getContent();
        $responseData = json_decode($content, true);
        $imageUrl     = $responseData['data'][0]['url'];

        return $imageUrl;
    }
}
