# Parte 1

## Ejercicio 1

Se definió el script `generar-compose.sh` que permite crear una definición de Docker Compose con una cantidad 
configurable de clientes (`<cantidad_clientes>`). Como se pidió, el nombre de los containers sigue el formato propuesto.

Para ejecutar el script, hacemos:

```sh
sh generar-compose.sh <nombre_del_archivo_de_salida> <cantidad_clientes>
```

Esto genera directamente el archivo de Docker Compose en el archivo de salida indicado.

## Ejercicio 2

Modifiqué el script `generar-compose.sh` para inyectar los archivos de configuración (`config.ini` y `config.yaml` respectivamente)
usando **Docker Volumes** para que no sea necesario copiar estos archivos dentro de la imagen y evitar el hecho de 
tener que reconstruírlas si queremos cambiar la configuración de los containers. Esto se logró simplemente agregando
en el cliente:

```yaml
volumes:
  - ./client/config.yaml:/config.yaml
```

y en el servidor:

```yaml
volumes:
  - ./server/config.ini:/config.ini
```

## Ejercicio 3

Para este caso, cree un script `validar-echo-server.sh` que permite verificar el correcto funcionamiento del servidor
(echo server) utilizando netcat para interactuar con el mismo. 

Primero, obtuve la configuración del server para enviarle el mensaje a esa IP y ese puerto:

```shell
HOST=$(grep "SERVER_IP" server/config.ini | cut -d'=' -f2 | xargs)
PORT=$(grep "SERVER_PORT" server/config.ini | cut -d'=' -f2 | xargs)
```

Luego, usé `docker run --rm` para correr un **contenedor efímero** (se borra al salir), el cual se conecta a la misma
red que el servidor (sin exponer puertos) usando:

```
--network=tp0_testing_net
```

La explicación de porqué elegimos esta red es porque, en el Docker Compose, definimos:

```yaml
networks:
 - testing_net
```

Entonces, cuando levantamos con `docker compose`, Docker **prefija el nombre de la red** con el `name:` del proyecto (que
en este caso es `tp0`). Por ende, como nuestro compose tiene `name: tp0`, la red `testing_net` se va a llamar realmente
**`tp0_testing_net`** en Docker.

Este contenedor va a correr la imagen `subfuzion/netcat`, que trae **netcat** instalado. La **única razón** para usar
`subfuzion/netcat` es que es una imagen mínima que ya trae **netcat instalado y listo para usar**.

`--entrypoint sh` → en lugar de usar el entrypoint default, corre un shell dentro del contenedor.

Ahora, como el **entrypoint** (qué comando corre al arrancar el contenedor) por defecto de la imagen que estamos usando
(`subfuzion/netcat`) es `nc` (netcat), y nosotros queremos correr **otro comando distinto** (en este caso, un `echo "msg" | nc ...`),
necesitamos poder ejecutar un shell (`sh`) dentro del contenedor. Para eso, ponemos:

```
--entrypoint sh
```

indicando que, en lugar de usar el entrypoint default, corra un shell dentro del contenedor.

Finalmente, el comando 

```shell
"echo \"$MSG\" | nc -w 20 $HOST $PORT"`
```

imprime el mensaje definido antes (`echo "$MSG"`) y lo manda al servidor usando `nc` (`nc -w 20 $HOST $PORT`, donde 
`-w 20` es un timeout de 20s), conectándose a `$HOST:$PORT` (el servidor echo), cuyos datos saca del `server/config.ini`.

Luego, lo que responda el servidor se guarda en `RESPONSE` y se hace el chequeo correspondiente para imprimir el resultado final.

Se implementó un validador del servidor, que para este entonces es un echo-server. 
Este programa `validar-echo-server.sh`, busca la coniguración del servidor para obtener su _ip_ y _puerto_ y levanta un
contenedor de docker corriendo la imágen de `subfuzion/netcat` que se conecta a la _network_ del servidor para hacer
una consulta y verificar que el servidor esté efectivamente levantado y listo para recibir consultas.

Este script se ejecuta haciendo:

```sh
sh validar-echo-server.sh
```

## Ejercicio 4

En este ejercicio le agregamos al servidor y al cliente la capacidad de manejar la señal **SIGTERM** para poder hacer
un **cierre graceful** al recibirla. Para eso, en el server agregué:

```py
signal.signal(signal.SIGTERM, sigterm_handler)
```

donde `sigterm_handler` es la función que se va a llamar cuando se reciba la señal, la cual se va a encargar de cerrar
los sockets y hacer los logs correspondientes.

Por otro lado, en el cliente agregué:

```go
sigs := make(chan os.Signal, 1)
signal.Notify(sigs, syscall.SIGTERM)
```

para crear un canal `sigs` de tipo `os.Signal` y definir que cualquier señal SIGTERM enviada al proceso se mande a este canal.

Luego, dentro del loop que manda los mensajes, agregué

```go
select {
case <-sigs:
    // Manejo de la señal
default:
    // Lógica normal del cliente
}
```

Entonces, si llega un valor en `sigs` (es decir, si el proceso recibe SIGTERM), se ejecuta el `case <-sigs`. Caso contrario,
se ejecuta default y sigue con el envío de mensajes al server.

Esto se puede verificar levantando los contenedores mostrando los logs:

```shell
make docker-compose-up; make docker-compose-logs
```

y, antes de que termine la ejecución normal, hacer:

```shell
make docker-compose-down
```

Esto mostrará lo siguiente en los logs:

```
client1  | 2025-09-04 18:51:40 INFO     action: shutdown | result: in_progress | client_id: 1 | msg: SIGTERM received
client1  | 2025-09-04 18:51:40 INFO     action: shutdown | result: success | client_id: 1
client1 exited with code 0                                                                                                                                  
server   | 2025-09-04 18:51:40 INFO     action: shutdown | result: in_progress | msg: SIGTERM received                                                      
server   | 2025-09-04 18:51:40 INFO     action: close_server_socket | result: success
server   | 2025-09-04 18:51:40 INFO     action: shutdown | result: success
server exited with code 0
```

indicando que se realizó el cierre correctamente.

# Parte 2

## Ejercicio 5

### Protocolo de comunicación

Para implementar la comunicación entre el cliente y el servidor en este ejercicio, el protocolo usado es el explicado
a continuación.

Para la comunicación, utilicé el siguiente y único mensaje:

| `BET_SIZE` | `BET` |
|------------|-------|
 
donde:

* `BET_SIZE` son 4 bytes que utilizan para comunicar el largo del payload del mensaje (`BET`) en bytes.
* `BET`: Es el payload, que representa los datos de la apuesta. 

La apuesta se representa con el siguiente struct:

```go
type BetMessage struct {
	Agency    string
	Firstname string
	Lastname  string
	Document  string
	Birthdate string
	Number    string
}
```

que, para ser enviado en el payload del mensaje, se le realiza un join de sus campos separándolos por comas, quedando el
payload de la forma:

```
Agency,Firstname,Lastname,Document,Birthdate,Number
```

Para evitar los short writes del lado del cliente, realizo la escritura de mensajes sobre el socket mediante un `bufio.Writer`,
que envuelve la conexión TCP. Esto permite acumular los bytes del mensaje en un buffer en memoria antes de enviarlos al socket,
en donde cada llamada a `socketWriter.Write` agrega los datos al buffer, y la llamada explícita a `socketWriter.Flush()`
es la que envía todos los bytes acumulados de forma segura y completa al socket, manejando short writes internamente y
garantizando que el receptor reciba el mensaje completo.

```go
socketWriter := bufio.NewWriter(s.conn)

socketWriter.Write(msgSizeBytes)
socketWriter.Write(msgBytes)

socketWriter.Flush()
```

Luego, desde el server, primero recibimos el largo de la apuesta haciendo:

```py
self._recv_all(BET_SIZE)
```

y luego, convirtiendo esos bytes en un entero, usamos ese tamaño para hacer un `recv` de la apuesta
basado en esa cantidad de bytes:

```py
bet_size = int.from_bytes(self._recv_all(BET_SIZE), "big")
bet_bytes = self._recv_all(bet_size)
```

El método `_recv_all(bytes_to_recv)` es el que permite evitar los **short reads**, ya que se va a seguir leyendo
del socket hasta que la cantidad de bytes recibidos sea igual a la cantidad de bytes esperados (`bet_size`). Si en algún
momento no se reciben más bytes pero la cantidad de bytes leídos es menor a la esperada, se levanta una excepción.

### Ejecución

Para ejecutar este ejercicio, hacemos como antes:

```sh
make docker-compose-up
```

## Ejercicio 6

### Protocolo de comunicación

Para implementar la comunicación en este ejercicio, modifiqué ligeramente el protocolo del ejercicio anterior, agregando
un byte más en el mensaje que indica si el batch enviado es el batch final o no. De esta forma, el mensaje nos queda:

| `FLAG_FINAL_BATCH` | `BATCH_SIZE` | `BATCH` |
|--------------------|--------------|---------|

donde:

* `FLAG_FINAL_BATCH`: Es 1 si el batch enviado es el batch final, 0 en caso contrario.
* `BATCH_SIZE` son 4 bytes que utilizan para comunicar el largo del payload del mensaje (`BATCH`) en bytes.
* `BATCH`: Es el payload, que representa un conjunto de apuestas (`BET`).

Internamente, el batch es un conjunto de apuestas (que tienen la misma representación que en el ejercicio anterior)
separadas por un punto y coma (`;`).

**Observación**: En este ejercicio tuve que usar un pequeño sleep (20ms) entre el envío de cada batch desde el cliente porque,
de no hacerlo, en un punto el servidor deja de leer los mensajes a pesar de que el cliente cierra bien (exit code 0) y 
que envía correctamente el flag del batch final. Cuando uso un csv más pequeño que el otorgado por la cátedra (por ejemplo,
el usado en los tests) no tengo este problema y el sleep no es necesario, solamente con archivos muy grandes. A pesar de
múltiples intentos en detectar cuál era el problema, no logré hacerlo y preferí dejar ese mínimo sleep (que no tiene grandes
implicancias en la performance) para poder obtener el resultado esperado y al hacer un graceful shutdown.

### Ejecución

Para la ejecución es igual que el ejercicio anterior.

## Ejercicio 7

En este ejercicio, tenemos que implementar un sorteo que solamente debe iniciar cuando todos los clientes terminen de
mandar sus batches de apuestas. Para poder lograr esto, agregué una variable de entorno `AMOUNT_CLIENTS` en el server,
que indica cuántos clientes esperamos que se conecten para enviar sus apuestas. Entonces, el server se va a quedar
esperando a que esa cantidad de clientes le envíen el nuevo mensaje: `notify_message`.

### Protocolo de comunicación

Como dije antes, ahora tenemos un mensaje nuevo (`notify_message`) y además otro mensaje (`id_message`) que sirve para
que el cliente le indique al servidor su ID (la cual usamos para guardarnos internamente el socket de comunicación de
ese cliente en el server y poder referirnos a él posteriormente).

Luego, el server envía el mensaje con los ganadores a los clientes (`winners_message`).

#### Batch Message

Es el mismo mensaje de antes, solo que ya no necesitamos el flag de batch final porque vamos a usar el nuevo mensaje
para indicar que el cliente terminó de mandar apuestas. Ahora, en su lugar, vamos a mandar otro flag (de un byte también)
que indica el tipo de mensaje que estamos enviando, usando **0** para el **Batch Message**.

Entonces, el mensaje nos queda:

| `MESSAGE_TYPE` | `BATCH_SIZE` | `BATCH` |
|----------------|--------------|---------|

donde:

* `MESSAGE_TYPE`: Indica que se trata de un mensaje de tipo **Batch Message** (siempre vale 0).
* `BATCH_SIZE` son 4 bytes que utilizan para comunicar el largo del payload del mensaje (`BATCH`) en bytes.
* `BATCH`: Es el payload, que representa un conjunto de apuestas (`BET`).

#### Notify Message

Este mensaje se usa para que los clientes notifiquen al servidor de que finalizaron con el envío de todas las apuestas 
y así proceder con el sorteo. En este caso, no tiene payload, solamente el tipo de mensaje, que es **1** para este mensaje.

Entonces, el mensaje nos queda:

| `MESSAGE_TYPE` |
|----------------|

#### ID Message

Este mensaje se usa para que el cliente le indique al servidor el número de ID. Simplemente tiene el `message type` (**2**)
y el ID en cuestión. Entonces, el mensaje es:

| `MESSAGE_TYPE` | `CLIENT_ID` |
|----------------|-------------|

donde:

* `MESSAGE_TYPE`: Indica que se trata de un mensaje de tipo **ID Message** (siempre vale 2).
* `CLIENT_ID`: Es el payload, que representa el ID del cliente. Siempre es de 1 byte.

#### Winners Message

Para que el server le pueda informar a los clientes los ganadores correspondientes, usamos un mensaje donde indicamos
inicialmente la cantidad de ganadores y posteriormente los ganadores en sí. 

Entonces, el primer mensaje nos queda:

| `AMOUNT_WINNERS` |
|------------------|

donde:

* `AMOUNT_WINNERS`: Son 4 bytes que indican la cantidad de ganadores en el mensaje. Únicamente se pasa el documento del ganador.

y el resto de mensajes son de la forma:

| `DOCUMENT_WINNER` |
|-------------------|

donde:

* `DOCUMENT_WINNER`: Es el payload de 4 bytes que almacenan el número de documento del ganador.

### Ejecución

Se ejecuta de la misma manera que los ejercicios anteriores.

# Parte 3

## Ejercicio 8

Para este ejercicio, modifiqué el servidor para que permita aceptar conexiones y procesar mensajes en paralelo.

Para ello, decidí utilizar el módulo `multiprocessing` para evitar el GIL (Global Interpreter Lock) de Python, el cual
hace que los threads no den paralelismo real de CPU. En definitiva, la función del GIL es asegurar que solo un thread de
Python ejecute código Python a la vez dentro de un mismo proceso. Entonces, aunque tengamos varios threads en Python,
nunca van a estar ejecutando código de Python puro al mismo tiempo en distintos núcleos.

Entonces, en mi caso, cada cliente aceptado no se maneja en un thread, sino en un proceso hijo independiente. 
Esto hace que cada proceso corra su "propia copia de Python". Así, por cada nueva conexión, el servidor crea un nuevo
`Process`:

```py
agency = Process(
            name=str(client.id),
            target=self.__handle_client_connection,
            args=(client, self._lock_bets_file, self._notify_barrier),
        )
```

A cada proceso le paso por argumento un `Lock` para el archivo de apuestas y una `Barrier` para sincronizar el inicio
del sorteo de apuestas.

El lock es necesario porque múltiples clientes van a querer acceder al archivo de apuestas, ya sea para leerlo (`load_bets`) o para 
escribirlo (`store_bets`). Entonces, definí el acceso al archivo (recurso compartido) como una sección crítica, y manejo
el acceso a dicha sección crítica y la sincronización entre procesos mediante dicho lock.

Por otro lado, la barrera la utilizo para sincronizar a todos los procesos en un punto común. Este punto común es justamnete
el momento previo al sorteo, donde los clientes tienen que esperar a que todos los demás clientes hayan terminado de 
mandar sus apuestas. Cuando el último cliente notifica dicha finalización, se termina esa espera, el server realiza el
sorteo y todos los clientes compiten por tomar el lock para poder acceder al archivo de apuestas y consultar sus ganadores.
Entonces, la barrerra es una herramienta de sincronización (no de comunicación) clave para poder realizar correctamente
este sorteo, y no iniciarlo antes de que alguno de los clientes termine de mandar sus apuestas (ni caer en un busy wait
para saber cuanddo terminaron de mandar).

### Ejecución

Se ejecuta de la misma manera que los ejercicios anteriores.