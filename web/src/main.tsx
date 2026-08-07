import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

// Utilities are imported before the project's own styles, so a CSS Module rule
// still wins over a utility when both target the same element.
import "virtual:uno.css";

import { App } from "./app/App";
import "./shared/styles/global.css";

const root = document.getElementById("root");

if (!root) {
  throw new Error("Etpeto root element was not found");
}

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
