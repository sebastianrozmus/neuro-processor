<?php

namespace App\Service;

use Symfony\Contracts\HttpClient\HttpClientInterface;
use Doctrine\ODM\MongoDB\DocumentManager;
use App\Document\GameGenerationLog;
use DateTime;

class OpenAiService // implements AiClientInterface
{
    public function __construct(private HttpClientInterface $client)
    {
    }

    public function call(string $prompt, string $gameId): string
    {
        $response = $this->client->request('POST', 'https://api.openai.com/v1/engines/davinci/completions', [
            'headers' => [
                'Authorization' => 'Bearer YOUR_API_KEY_HERE',
                'Content-Type' => 'application/json',
            ],
            'json' => [
                'prompt' => $prompt,
                'max_tokens' => 100,
            ],
        ]);

        $content      = $response->getContent();
        $responseData = json_decode($content, true);
        $textResponse = $responseData['choices'][0]['text'] ?? 'No response';

        // TODO: AiResponseEvent: save costs, log to mongodb

        return $textResponse;
    }
}