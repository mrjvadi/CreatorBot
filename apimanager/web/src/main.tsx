import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "react-hot-toast";
import { useTranslation } from "react-i18next";
import "./i18n";
import App from "./App";
import "./index.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 15_000,
    },
  },
});

function LocalizedToaster() {
  const { i18n } = useTranslation();
  return (
    <Toaster
      position="top-center"
      toastOptions={{
        className: "font-sans text-sm",
        style: { direction: i18n.dir() },
      }}
    />
  );
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
        <LocalizedToaster />
      </BrowserRouter>
    </QueryClientProvider>
  </React.StrictMode>
);
