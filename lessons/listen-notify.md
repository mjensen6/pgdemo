---
title: LISTEN and NOTIFY
tldr: One session sends a message and every session listening gets it at once, with no polling, and only when the sender's transaction commits.
---

```setup
CREATE TABLE inbox (id serial PRIMARY KEY, at timestamptz DEFAULT now(), channel text, payload text);
CREATE TABLE orders (id serial PRIMARY KEY, item text);
CREATE TABLE shipments (id serial PRIMARY KEY, order_id int);
-- The check waits a moment for the listener to record a message.
CREATE FUNCTION inbox_has(want_channel text) RETURNS bool LANGUAGE plpgsql AS $$
BEGIN
  FOR i IN 1..20 LOOP
    IF EXISTS (SELECT FROM inbox WHERE channel = want_channel) THEN RETURN true; END IF;
    PERFORM pg_sleep(0.25);
  END LOOP;
  RETURN false;
END $$;
```

```view
SELECT to_char(at, 'HH24:MI:SS') AS received, channel, payload FROM inbox ORDER BY id DESC LIMIT 8;
```

```cleanup
\! kill "$(cat /tmp/pgdemo-listener.pid 2>/dev/null)" 2>/dev/null; rm -f /tmp/pgdemo-listener.pid
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE application_name = 'pgdemo-listener';
```

Often one part of a system needs to know the moment something changes: a new order arrived, so the warehouse screen should update and a worker should start packing. The simple way is **polling**: ask "anything new?" every second. That's slow to notice, and it wastes work when nothing has changed.

Postgres has messaging built in:

- **`LISTEN orders`** subscribes the current session to the channel `orders`. A channel is just a name, and you don't create it first.
- **`NOTIFY orders, 'order 42'`** sends a message, the **payload**, to every session listening on `orders`. `pg_notify('orders', text)` does the same as a function, which is handy when the payload is computed.

Listeners get the message immediately, with no polling.

**Notifications are transactional.** A `NOTIFY` inside a transaction is sent only when the transaction **commits**. If it rolls back, the message is never sent. So listeners never hear about changes that didn't happen. That makes a **trigger** a natural place to send one: every insert notifies, and only committed inserts are heard.

**The limits.** It's not a queue. A session that isn't listening at that moment misses the message for good, and nothing is stored. Payloads are limited to about 8,000 bytes, so send an ID and let the listener look up the rest. For work that must not be lost, store it in a table, like the SKIP LOCKED lesson, and use NOTIFY to wake workers up.

**Where it came from.** LISTEN and NOTIFY come from the Berkeley POSTGRES project, so they're older than the name PostgreSQL. Payloads were added in Postgres 9.0 (2010). Before that, a notification carried only its channel name. Job libraries like River (lesson 22) use it to wake workers the instant a job is added.

**A listener for this lesson.** psql isn't a good listener on its own, since it only shows notifications when you run a command. So the first step starts a small background listener that records every message on `orders` and `shipments` into table `inbox`.

**The Live View** shows table `inbox`, newest first: every message the listener has received.

## Try it

Start the listener in the background. It runs for 20 minutes, or until you reset the lesson.

```sh
[ -f /tmp/pgdemo-listener.pid ] && kill "$(cat /tmp/pgdemo-listener.pid)" 2>/dev/null
cat > /tmp/pgdemo-listener.sh <<'SH'
{ echo "LISTEN orders; LISTEN shipments;"; for _ in $(seq 6000); do echo "SELECT;"; sleep 0.2; done; } |
  PGAPPNAME=pgdemo-listener psql -X -q -At 2>&1 |
  sed -un 's/^Asynchronous notification "\(.*\)" with payload "\(.*\)" received from server process.*/\1\t\2/p' |
  while IFS=$'\t' read -r ch pl; do
    printf '%s\n' "INSERT INTO inbox (channel, payload) VALUES (:'ch', :'pl');" | psql -X -q -v ch="$ch" -v pl="$pl"
  done
SH
nohup bash /tmp/pgdemo-listener.sh >/dev/null 2>&1 & echo $! > /tmp/pgdemo-listener.pid
sleep 1; psql -Atc "SELECT 'Listener running: ' || count(*) FROM pg_stat_activity WHERE application_name = 'pgdemo-listener'"
```

Send a message. It appears in the Live View within a moment.

```sql
NOTIFY orders, 'hello from the SQL session';
```

Notifications wait for commit. The first one is rolled back, so it's never sent. The second is sent only when `COMMIT` runs. Watch the Live View: only the committed message arrives.

```sql
BEGIN;
NOTIFY orders, 'this one is rolled back';
ROLLBACK;
BEGIN;
NOTIFY orders, 'this one is committed';
COMMIT;
```

Notify from a trigger, so every new order announces itself with its ID. `pg_notify` takes the payload as text, so the new row's id is cast.

```sql
CREATE FUNCTION announce_order() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  PERFORM pg_notify('orders', 'new order ' || NEW.id);
  RETURN NEW;
END $$;
CREATE TRIGGER orders_announce AFTER INSERT ON orders FOR EACH ROW EXECUTE FUNCTION announce_order();
INSERT INTO orders (item) VALUES ('lamp'), ('tent');
```

## Challenge

Make `shipments` announce itself too: create a trigger so every insert into `shipments` sends a message on the channel `shipments`, then insert a shipment. The listener is already listening on that channel.

Hint: Follow the last step: a new function that calls pg_notify('shipments', ...), a trigger on shipments, then an INSERT INTO shipments (order_id) VALUES (1).

```solution
CREATE FUNCTION announce_shipment() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  PERFORM pg_notify('shipments', 'shipment ' || NEW.id);
  RETURN NEW;
END $$;
CREATE TRIGGER shipments_announce AFTER INSERT ON shipments FOR EACH ROW EXECUTE FUNCTION announce_shipment();
INSERT INTO shipments (order_id) VALUES (1);
```

```check
SELECT inbox_has('shipments');
```
