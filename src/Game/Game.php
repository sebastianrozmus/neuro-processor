<?php

namespace App\Game;

use React\EventLoop\Factory as LoopFactory;
use React\Stream\ReadableResourceStream;

class Game
{
    private Scene $actualScene;

    private SceneManager $sceneManager;

    public function __construct(
        private $output,
    ) {
        // code...
    }

    public function initialize()
    {
        $this->setConsoleMode();
        $this->registerShutdownFunction();
        $scene = 'menu';

        $this->loop        = LoopFactory::create();
        $this->stdinStream = new ReadableResourceStream(STDIN, $this->loop, -1);
        // $this->actualScene = Scene::MENU;

        $output = $this->output;
        $this->stdinStream->on('data', function ($data) use (&$name, &$scene, $output) {
            $data = trim($data);
            echo "Raw data: " . bin2hex($data) . PHP_EOL;


            switch ($scene) {
                case 'menu':
                    $name .= $data;
                    $this->output->write($data);
                    if (0 === ord($data)) {
                        //$name = $data;
                        $scene = 'game'; // Change scene to game
                        $this->output->writeln("Welcome to the game, $name! Press Shift+Enter to greet.");
                    }
                    break;

                case 'game':
                    var_dump($data);
                    var_dump(trim($data));
                    $this->output->writeln(mb_ord($data));
                    if ($data === "\x0D") { // This is an attempt to catch Shift+Enter, but it's actually Enter.
                        $this->output->writeln("Hello, $name!");
                    }
                    break;
            }
        });

        $this->loop->run();
    }

    public function setConsoleMode()
    {
        system('stty cbreak -echo');
    }

    private function registerShutdownFunction()
    {
        register_shutdown_function(function () {
            system('stty sane');
        });
    }
}
