# Container usage

Running Disco inside a container is recommended for increased security. It allows storing the encryption key as a secret, which protects against exposing it to the shell history and other running host processes.

The following are instructions for setting up this workflow.

1. Install [Podman](https://podman.io/docs/installation) and [podman-compose](https://github.com/containers/podman-compose).

> [!NOTE]
> While this workflow can be done with Docker, we recommend using Podman instead.
> The secret functionality in Docker requires enabling Swarm mode, which is not
> required (or supported) in Podman. Podman can also be used without root
> permissions, though running it as root is an additional layer of security,
> since it requires typing the sudo password.

2. Pull the image of the latest stable version of Disco:
   ```sh
   podman pull docker.io/hackfixme/disco:latest
   ```

3. Initialize Disco, creating a Podman volume so that the data is persisted between runs, and at the same time create a Podman secret from the output:
   ```sh
   podman run --rm -it --volume disco:/opt/disco hackfixme/disco:latest init \
   | tee /dev/tty | head -1 | sed 's:.* ::' | podman secret create disco_key -
   ```

   You can also manually copy the key and create a secret, but this way avoids storing
   the key in your clipboard, or risking the key being stored in your shell history,
   as explained in the warning [here](./get_started.md#setting-the-encryption-key).

4. Clone the Disco repository, or copy the [`docker-compose.yml` file](https://github.com/hackfixme/disco/blob/main/docker-compose.yml) locally, and run:
   ```sh
   podman-compose up
   ```

   This will create the pod needed for subsequent commands. This only needs to be run once.

5. Run Disco commands as usual, e.g.:
   ```sh
   podman-compose run --rm disco set key value
   ```

   You can set this as an alias for convenience:
   ```sh
   alias discon='podman-compose run --rm disco'
   ```

   Note that if you want to run the web server, you might want to pass the `--service-ports` option to `podman-compose`, in order to expose the server ports to the host network. So the full command should be:
   ```sh
   podman-compose run --rm --service-ports disco serve
   ```
