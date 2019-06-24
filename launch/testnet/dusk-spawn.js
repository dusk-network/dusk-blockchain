#!/usr/local/bin/node

const { spawn } = require("child_process");

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

async function stdout({ stdout }, id) {
  for await (const line of lines(stdout)) {
    console.log(`spawner ${id}: ${line}`);
  }
}

function main(nodes = 1) {
  for (let i = 0; i < +nodes; i++) {
    const port = 7001 + i;
    const child = spawn(
      "./testnet",
      ["-p=" + port, "-d=demo" + port],
      { stdio: ["ignore", "pipe", process.stderr] }
    );

    stdout(child, port);
  }
}
main(process.argv[2]);
