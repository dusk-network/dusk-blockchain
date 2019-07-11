#!/usr/local/bin/node

const path = require("path");
const { spawn } = require("child_process");
const { mkdir } = require("fs").promises;

const basepath = require("os").tmpdir();

var isLinux = process.platform === "linux";

async function* lines(chunks) {
  let previous = "";
  for await (const chunk of chunks) {
    previous += chunk;
    let eolIndex;
    while ((eolIndex = previous.indexOf("\n")) >= 0) {
      yield previous.slice(0, eolIndex);
      previous = previous.slice(eolIndex + 1);
    }
  }
  yield previous;
}

async function stdout(prefix, readable) {
  for await (const line of lines(readable)) {
    console.log(`${prefix}: ${line}`);
  }
}

async function main(nodes = 1, ...flags) {

  blindBid = isLinux ? "./blindbid-avx2" : "./blindbid-mac"

  for (let i = 0; i < +nodes; i++) {
    const port = 7000 + i;
    const rpcport = 9000 + i;

    process.env["TMPDIR"] = path.join(basepath, "nodes", String(port));
    await mkdir(process.env["TMPDIR"], { recursive: true });

    const node = spawn(
      "./testnet",
      ["-p=" + port, "-d=demo" + port, "-r=" + rpcport,],
      { stdio: ["ignore", "pipe", "pipe" ]}
    );

    stdout(`spawner ${port}`, node.stdout);
    stdout(`spawner ${port}`, node.stderr);

    const bid = spawn(blindBid, flags, {
      stdio: ["ignore", "pipe", "pipe"]
    });
    stdout(`bid ${port}`, bid.stdout);
    stdout(`bid ${port}`, bid.stderr);
    
  }
}
main(process.argv.slice(2));
