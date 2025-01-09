
import * as readline from "node:readline";
import {stdin, stdout} from "node:process";
const __dirname = import.meta.dirname;
import fs from "node:fs";

const rl = readline.createInterface({input: stdin, terminal: false});
const logPath = __dirname + "/log";
function say(obj){
    fs.writeSync(logPath, JSON.stringify(obj) + "\n");
}

rl.on("line", (l) => {
    let obj = JSON.parse(l);
    if(obj.event == "init"){
        // Ack
        say({});
    }else if(obj.event == "terminate"){
        // Do nothing, wait for termination
    }else if(obj.event == "upload"){
        say({ "event": "complete", "oid":obj.oid, "error": {"code":1, "message": "Upload is not supported"}});
    }else if(obj.event == "download"){
        say({ "event": "complete", "oid":obj.oid, "error": {"code":1, "message": JSON.stringify(obj)}});
    }
});