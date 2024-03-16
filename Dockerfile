FROM php:8.3

RUN apt-get update && \
    apt-get install -y \
        git \
        curl \
        vim \
        libncurses5-dev \
        ncurses-doc \
        libncursesw5-dev \
        zlib1g-dev \
        wget \
        libzip-dev \
    && apt-get clean

RUN docker-php-ext-install zip 

RUN pecl install mongodb \
    &&  echo "extension=mongodb.so" > $PHP_INI_DIR/conf.d/mongo.ini

RUN apt-get update && apt-get install -y wget tar

WORKDIR /usr/src/app

RUN wget https://github.com/TheZoraiz/ascii-image-converter/releases/download/v1.13.1/ascii-image-converter_Linux_arm64_64bit.tar.gz

RUN tar -xzf ascii-image-converter_Linux_arm64_64bit.tar.gz && \
    mv ascii-image-converter_Linux_arm64_64bit/ascii-image-converter /usr/local/bin/ && \
    chmod +x /usr/local/bin/ascii-image-converter

RUN rm -rf ascii-image-converter_Linux_arm64_64bit.tar.gz ascii-image-converter_Linux_arm64_64bit

RUN php -r "copy('https://getcomposer.org/installer', 'composer-setup.php');" \
    && php -r "if (hash_file('SHA384', 'composer-setup.php') === 'dac665fdc30fdd8ec78b38b9800061b4150413ff2e3b6f88543c636f7cd84f6db9189d43a81e5503cda447da73c7e5b6') { echo 'Installer verified'; } else { echo 'Installer corrupt'; unlink('composer-setup.php'); } echo PHP_EOL;" \
    && php composer-setup.php \
    && mv composer.phar /usr/local/bin/composer \
    && php -r "unlink('composer-setup.php');"

RUN apt-get install -y git

RUN cd ~ && git clone https://github.com/beekmanbv/mod_ncurses

RUN cd ~/mod_ncurses && \
    phpize && \
    ./configure && \
    make && \
    make install && \
    docker-php-ext-enable ncurses && \
    rm -Rf ~/ncurses ~/channels.xml

WORKDIR /app

CMD ["tail", "-f", "/dev/null"]
