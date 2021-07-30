# Setup purgeman systemd service

Copy the purgeman binary `bin/purgeman` to `/usr/bin/`.

Copy the systemd service `purgeman.service` to `/usr/lib/systemd/system/`.

Create a service user `purgeman`.
```bash
sudo adduser -r -d /dev/null -s /sbin/nologin purgeman
```

Copy the purgeman configuration `purgeman.conf` to `/etc/purgeman/`.
Be sure that this file must be only accessible by the `purgeman` user.

Start the service.
```bash
sudo service purgeman start
```