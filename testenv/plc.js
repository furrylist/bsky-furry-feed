// This file is largely vibecoded to just have a mock PLC server.
// It's just a glorified array over the network.
import { createServer } from "http";
import plc from "/app/node_modules/.pnpm/@did-plc+lib@0.0.4/node_modules/@did-plc/lib/dist/index.js";
import { cidForCbor } from "/app/node_modules/.pnpm/@did-plc+lib@0.0.4/node_modules/@atproto/common/dist/index.js";

const store = {};

async function validateAndAdd(did, op) {
  const ops = store[did] || [];
  const { nullified } = await plc.assureValidNextOp(did, ops, op);
  const cid = await cidForCbor(op);
  store[did] = store[did] || [];
  store[did].push({
    did,
    operation: op,
    cid,
    nullified: false,
    createdAt: new Date(),
  });
  for (const nullCid of nullified) {
    for (const entry of store[did]) {
      if (nullCid.equals(entry.cid)) entry.nullified = true;
    }
  }
}

function lastOp(did) {
  const ops = (store[did] || []).filter((o) => !o.nullified);
  return ops.length ? ops[ops.length - 1].operation : null;
}

const server = createServer(async (req, res) => {
  const url = new URL(req.url, "http://localhost");
  const parts = url.pathname.split("/").filter(Boolean);
  const did = parts[0] ? decodeURIComponent(parts[0]) : null;
  try {
    if (req.method === "GET" && url.pathname === "/_health") {
      res.writeHead(200);
      res.end(JSON.stringify({ version: "0.0.0" }));
      return;
    }
    if (req.method === "POST" && did) {
      let body = "";
      for await (const chunk of req) body += chunk;
      await validateAndAdd(did, JSON.parse(body));
      res.writeHead(200);
      res.end();
      return;
    }
    if (req.method === "GET" && did) {
      const op = lastOp(did);
      if (!op) {
        res.writeHead(404);
        res.end();
        return;
      }
      const data = plc.opToData(did, op);
      if (!data) {
        res.writeHead(404);
        res.end();
        return;
      }
      const doc = await plc.formatDidDoc(data);
      res.writeHead(200, { "content-type": "application/did+ld+json" });
      res.end(JSON.stringify(doc));
      return;
    }
    res.writeHead(404);
    res.end();
  } catch (err) {
    res.writeHead(400);
    res.end(JSON.stringify({ error: err.message }));
  }
});

const port = parseInt(process.env.PLC_PORT || "2582", 10);
server.listen(port, "0.0.0.0", () => process.stdout.write("plc ready\n"));
