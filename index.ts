import http from "http";
import { CronJob } from "cron";
import asciiArt from "./logo";
import fs from "fs/promises";
import path from "path";

const BASE_URL = (process.env.BASE_URL ?? "http://127.0.0.1").replace(/\/$/, "");
const PORT = Number(process.env.PORT ?? "3000");
const ALLOW_INSECURE_TLS = process.env.ALLOW_INSECURE_TLS === "true";

if (ALLOW_INSECURE_TLS) {
  // Opt in only when you actually need to talk to a self-signed HTTPS endpoint.
  process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
}

// Display state interface
interface DisplayState {
  visible: boolean;
  lastUpdated: number;
  lastAction: string;
}

// In-memory state store
let displayState: DisplayState = {
  visible: false,
  lastUpdated: Date.now(),
  lastAction: "initialized",
};

// Initialize display state
function initializeState() {
  console.log("Display state initialized in memory");
}

// Get current display state
function getDisplayState(): DisplayState {
  return displayState;
}

// Update display state
function updateDisplayState(visible: boolean, action: string) {
  displayState = {
    visible,
    lastUpdated: Date.now(),
    lastAction: action,
  };
  console.log(
    `Display state updated: ${visible ? "visible" : "hidden"} (${action})`
  );
}

async function setDisplayClock({ value }: { value: boolean }) {
  const timestamp = Date.now();
  const ENDPOINT = `${BASE_URL}/hh?command=setDisplayXclock&value=${value}&_=${timestamp}`;
  try {
    const response = await fetch(ENDPOINT, {
      method: "GET",
      headers: {
        Accept: "text/plain, */*; q=0.01",
        "X-Requested-With": "XMLHttpRequest",
      },
    });

    if (!response.ok) {
      if (response.status === 503) {
        console.warn("Received 503, retrying in 5 seconds...");
        await new Promise((resolve) => setTimeout(resolve, 5000));
        return await setDisplayClock({ value });
      }
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    await response.text();
    console.log(`Display clock set to ${value}`);

    // Update state in memory after successful API call
    updateDisplayState(value, `manual_set_${value ? "true" : "false"}`);
  } catch (error) {
    console.error("Error:", error);
  }
}

async function showDisplayClock() {
  await setDisplayClock({ value: true });
  updateDisplayState(true, "manual_show");
}
async function hideDisplayClock() {
  await setDisplayClock({ value: false });
  updateDisplayState(false, "manual_hide");
}

// Every day at 22:00 (10 PM)
new CronJob(
  "0 22 * * *",
  async () => {
    await showDisplayClock();
    updateDisplayState(true, "cron_show_22:00");
  },
  null,
  true,
  "Europe/Zurich"
);

// Every day at 06:00 (6 AM)
new CronJob(
  "0 6 * * *",
  async () => {
    await hideDisplayClock();
    updateDisplayState(false, "cron_hide_06:00");
  },
  null,
  true,
  "Europe/Zurich"
);

// Simple HTTP API for manual toggling
const server = http.createServer(async (req, res) => {
  const url = new URL(req.url || "", `http://${req.headers.host}`);

  // Serve static files and index.html for frontend, but keep API routes available.
  if (req.method === "GET") {
    const apiPrefixes = ["/toggle", "/show", "/hide", "/cron"];
    const isApi = apiPrefixes.some(
      (p) => url.pathname === p || url.pathname.startsWith(p + "?")
    );

    if (!isApi) {
      const publicRoot = path.resolve(process.cwd(), "public");
      let relPath = url.pathname;
      // Always serve index.html for root
      if (relPath === "/") relPath = "/index.html";
      // prevent directory traversal
      relPath = path.normalize(relPath).replace(/^\.+/, "");
      const filePath = path.join(publicRoot, relPath);

      // simple mime types
      const ext = path.extname(filePath).toLowerCase();
      const mime: Record<string, string> = {
        ".html": "text/html; charset=utf-8",
        ".js": "application/javascript; charset=utf-8",
        ".css": "text/css; charset=utf-8",
        ".png": "image/png",
        ".jpg": "image/jpeg",
        ".svg": "image/svg+xml",
        ".json": "application/json",
      };

      try {
        const data = await fs.readFile(filePath);
        res.writeHead(200, {
          "Content-Type": mime[ext] || "application/octet-stream",
        });
        res.end(data);
        return;
      } catch (err) {
        // If root and index.html missing, show clear error
        if (url.pathname === "/") {
          res.writeHead(500, { "Content-Type": "text/plain" });
          res.end("Error: public/index.html not found");
          return;
        }
        // else fall through to API/404
      }
    }
  }

  if (req.method === "GET") {
    if (url.pathname === "/toggle") {
      const valueParam = url.searchParams.get("value");
      if (valueParam === "true" || valueParam === "false") {
        const value = valueParam === "true";
        await setDisplayClock({ value });
        updateDisplayState(value, `manual_toggle_${value ? "true" : "false"}`);
        res.writeHead(200, { "Content-Type": "text/plain" });
        res.end(`Display clock set to ${value}`);
        return;
      } else {
        res.writeHead(400, { "Content-Type": "text/plain" });
        res.end("Missing or invalid 'value' parameter. Use true or false.");
        return;
      }
    }

    if (url.pathname === "/show") {
      await showDisplayClock();
      res.writeHead(200, { "Content-Type": "text/plain" });
      res.end("Display clock shown");
      return;
    }

    if (url.pathname === "/hide") {
      await hideDisplayClock();
      res.writeHead(200, { "Content-Type": "text/plain" });
      res.end("Display clock hidden");
      return;
    }

    if (url.pathname === "/cron") {
      res.writeHead(200, { "Content-Type": "text/plain" });
      res.end(
        "Cron jobs:\n" +
          "- Show clock: every day at 22:00 (0 22 * * *)\n" +
          "- Hide clock: every day at 06:00 (0 6 * * *)"
      );
      return;
    }

    // New endpoint to get current display state
    if (url.pathname === "/state") {
      const state = getDisplayState();
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify(state));
      return;
    }
  }

  res.writeHead(404, { "Content-Type": "text/plain" });
  res.end("Not found");
});

if (!Number.isFinite(PORT) || PORT <= 0) {
  throw new Error("PORT must be a positive number");
}

// Initialize state before starting server
initializeState();

server.listen(PORT, "0.0.0.0", () => {
  console.log("------------------------------------------------------------");
  console.log(asciiArt);
  console.log("------------------------------------------------------------");
  console.log(`Toggle clock: http://localhost:${PORT}/toggle?value=true|false`);
  console.log(`Show clock: http://localhost:${PORT}/show`);
  console.log(`Hide clock: http://localhost:${PORT}/hide`);
  console.log(`Get state: http://localhost:${PORT}/state`);
  console.log(`Cron job: http://localhost:${PORT}/cron`);
  console.log("------------------------------------------------------------");
  console.log(`Server running at http://localhost:${PORT}`);
  console.log("------------------------------------------------------------");
});
