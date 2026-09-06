function copyFallback(value: string) {
  try {
    const area = document.createElement("textarea");
    area.value = value;
    area.setAttribute("readonly", "");
    area.style.position = "fixed";
    area.style.top = "-9999px";

    document.body.appendChild(area);
    area.select();

    const copied = document.execCommand("copy");
    document.body.removeChild(area);

    return copied;
  } catch {
    return false;
  }
}

export async function copyText(value: string) {
  if (typeof navigator !== "undefined" && navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(value);
      return true;
    } catch {
      return copyFallback(value);
    }
  }

  return copyFallback(value);
}
