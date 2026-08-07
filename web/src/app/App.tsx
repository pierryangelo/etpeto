import { EditorWorkspace } from "../features/editor/EditorWorkspace";
import { LoginPage } from "../features/auth/LoginPage";

export function App() {
  if (window.location.pathname === "/login") {
    return <LoginPage />;
  }
  return <EditorWorkspace />;
}
