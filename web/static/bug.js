const tokenKey = "sovereign_token";

function getToken() {
  return localStorage.getItem(tokenKey) || "";
}

function setMsg(el, text, bad=false) {
  el.textContent = text;
  el.classList.toggle("bad", !!bad);
}

async function submitBugReport(subject, body, files) {
  const token = getToken();
  if (!token) throw new Error("You must be logged in in the main game window before submitting a bug report.");

  const fd = new FormData();
  fd.append("subject", subject);
  fd.append("body", body);
  for (const f of files) {
    fd.append("attachments", f, f.name);
  }

  const res = await fetch("/api/bug_report", {
    method: "POST",
    headers: { "Authorization": "Bearer " + token },
    body: fd,
  });
  let data = {};
  try { data = await res.json(); } catch (_) {}
  if (!res.ok) {
    throw new Error((data && (data.error || data.message)) ? (data.error || data.message) : "Request failed");
  }
  return data;
}

document.addEventListener("DOMContentLoaded", () => {
  const form = document.getElementById("bugForm");
  const subjectEl = document.getElementById("bugSubject");
  const bodyEl = document.getElementById("bugBody");
  const filesEl = document.getElementById("bugFiles");
  const msgEl = document.getElementById("bugMsg");
  const hintEl = document.getElementById("bugHint");

  if (!getToken()) {
    hintEl.textContent = "Not logged in. Open the main game window, log in, then return here to submit.";
  }

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    setMsg(msgEl, "", false);

    const subject = subjectEl.value.trim();
    const body = bodyEl.value.trim();
    if (!subject || !body) {
      setMsg(msgEl, "Title and description are required.", true);
      return;
    }

    const files = filesEl.files ? Array.from(filesEl.files) : [];
    try {
      await submitBugReport(subject, body, files);
      setMsg(msgEl, "Bug report sent to admin.", false);
      bodyEl.value = "";
      filesEl.value = "";
    } catch (err) {
      setMsg(msgEl, err.message || String(err), true);
    }
  });
});
