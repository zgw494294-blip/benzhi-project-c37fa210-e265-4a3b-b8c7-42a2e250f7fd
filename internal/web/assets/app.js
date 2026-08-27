let project = null;
let workspace = null;

const $ = (selector) => document.querySelector(selector);
const idempotencyKey = () => crypto.randomUUID();

async function api(path, method = "GET", body) {
  const response = await fetch(path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await response.json();
  if (!response.ok) {
    const conflicts = data.conflicts ? `\n${JSON.stringify(data.conflicts)}` : "";
    throw new Error(`${data.error}${conflicts}`);
  }
  return data;
}

function showError(error) {
  $("#status").textContent = error.message;
  $("#status").className = "warn";
}

function versioned(actor) {
  if (!workspace) {
    throw new Error("请先选择项目");
  }
  return {
    actor,
    expectedVersion: workspace.project.version,
    idempotencyKey: idempotencyKey(),
  };
}

function appendOption(select, value, text) {
  const option = document.createElement("option");
  option.value = value;
  option.textContent = text;
  select.append(option);
}

async function loadProjects() {
  const items = await api("/api/projects");
  const select = $("#projects");
  select.replaceChildren();
  appendOption(select, "", "选择已有项目");
  items.forEach((item) => appendOption(select, item.id, `${item.title} · ${item.status}`));
}

function fillRules(item) {
  ["title", "sourceLanguage", "targetLanguage", "frameRate", "minDisplayMillis", "maxDisplayMillis"]
    .forEach((key) => {
      $(`#rules [name=${key}]`).value = item[key];
    });
}

function renderTerms(terms) {
  const list = $("#terms");
  list.replaceChildren();
  if (terms.length === 0) {
    const empty = document.createElement("li");
    empty.textContent = "尚未登记术语";
    list.append(empty);
    return;
  }
  terms.forEach((term) => {
    const item = document.createElement("li");
    item.textContent = `${term.sourceText} → ${term.requiredTranslation}（v${term.version}）`;
    list.append(item);
  });
}

function renderFindings(items) {
  const target = $("#findings");
  target.replaceChildren();
  if (items.length === 0) {
    target.textContent = "没有匹配的问题";
    return;
  }
  items.forEach((finding) => {
    const label = document.createElement("label");
    label.className = "finding";
    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.dataset.finding = finding.id;
    checkbox.disabled = finding.status !== "open";
    const resolution = finding.resolutionNote ? `（${finding.resolutionNote}）` : "";
    label.append(checkbox, `${finding.ruleCode} · cue ${finding.cueSequence} · ${finding.status} · ${finding.message}${resolution}`);
    target.append(label);
  });
}

function showDownloads() {
  $("#downloads").hidden = false;
  document.querySelectorAll("[data-kind]").forEach((link) => {
    link.href = `/api/projects/${project.id}/exports/${link.dataset.kind}`;
  });
}

async function loadWorkspace() {
  if (!project) return;
  workspace = await api(`/api/projects/${project.id}`);
  const item = workspace.project;
  $("#status").textContent = `${item.title} · ${item.status} · v${item.version}`;
  $("#status").className = "";
  fillRules(item);
  $("#validationHint").textContent = item.currentRevisionID && !workspace.validation
    ? "规则或内容已变化，送审前必须重新校验"
    : "";
  $("#glossaryVersion").textContent = `当前术语版本：${item.glossaryVersion}`;
  renderTerms(workspace.terms);
  $("#revisions").textContent = workspace.revisions.length
    ? workspace.revisions.map((revision) => `#${revision.revisionNumber} ${revision.summary || "无摘要"} · ${revision.submittedBy} · ${revision.id}`).join("\n")
    : "尚无修订";
  renderFindings(workspace.validation ? workspace.validation.findings : []);
  if (workspace.manifest) {
    showDownloads();
    $("#freezePreview").textContent = JSON.stringify(workspace.manifest, null, 2);
  }
}

async function act(path, body, method = "POST") {
  if (!project) throw new Error("请先选择项目");
  const data = await api(`/api/projects/${project.id}${path}`, method, body);
  await loadWorkspace();
  return data;
}

$("#create").addEventListener("submit", async (event) => {
  event.preventDefault();
  try {
    const command = Object.fromEntries(new FormData(event.target));
    ["frameRate", "minDisplayMillis", "maxDisplayMillis"].forEach((key) => {
      command[key] = Number(command[key]);
    });
    Object.assign(command, { actor: "负责人", idempotencyKey: idempotencyKey() });
    project = await api("/api/projects", "POST", command);
    await loadProjects();
    $("#projects").value = project.id;
    await loadWorkspace();
  } catch (error) {
    showError(error);
  }
});

$("#projects").addEventListener("change", async (event) => {
  if (!event.target.value) return;
  project = { id: event.target.value };
  try {
    await loadWorkspace();
  } catch (error) {
    showError(error);
  }
});

$("#rules").addEventListener("submit", async (event) => {
  event.preventDefault();
  try {
    const command = Object.fromEntries(new FormData(event.target));
    ["frameRate", "minDisplayMillis", "maxDisplayMillis"].forEach((key) => {
      command[key] = Number(command[key]);
    });
    Object.assign(command, versioned("负责人"));
    await act("/rules", command, "PUT");
  } catch (error) {
    showError(error);
  }
});

$("#importTerms").addEventListener("click", async () => {
  try {
    await act("/terms/batch", {
      entries: JSON.parse($("#termBatch").value),
      ...versioned("译员"),
    });
  } catch (error) {
    showError(error);
  }
});

function parseCues() {
  return $("#cues").value.split(/\n/).filter(Boolean).map((line) => {
    const [sequence, inMillis, outMillis, sourceText, translatedText, speaker] = line.split("|");
    return {
      sequence: Number(sequence),
      inMillis: Number(inMillis),
      outMillis: Number(outMillis),
      sourceText,
      translatedText,
      speaker,
    };
  });
}

$("#submitRevision").addEventListener("click", async () => {
  try {
    await act("/revisions", {
      submittedBy: "译员",
      summary: $("#revisionSummary").value,
      cues: parseCues(),
      ...versioned("译员"),
    });
  } catch (error) {
    showError(error);
  }
});

$("#showDiff").addEventListener("click", async () => {
  try {
    const revisions = workspace.revisions;
    if (revisions.length < 2) throw new Error("当前没有可比较的父子修订");
    const from = revisions[revisions.length - 2].id;
    const to = revisions[revisions.length - 1].id;
    const diff = await api(`/api/projects/${project.id}/revisions/diff?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`);
    $("#diff").textContent = JSON.stringify(diff, null, 2);
  } catch (error) {
    showError(error);
  }
});

$("#validate").addEventListener("click", async () => {
  try {
    await act("/validate", versioned("译员"));
  } catch (error) {
    showError(error);
  }
});

$("#filterFindings").addEventListener("click", async () => {
  try {
    const query = new URLSearchParams({
      severity: $("#severity").value,
      ruleCode: $("#ruleCode").value,
      status: $("#findingStatus").value,
    });
    renderFindings(await api(`/api/projects/${project.id}/findings?${query}`));
  } catch (error) {
    showError(error);
  }
});

$("#resolveBatch").addEventListener("click", async () => {
  try {
    const note = $("#resolutionNote").value;
    const items = [...document.querySelectorAll("[data-finding]:checked")].map((element) => ({
      findingID: element.dataset.finding,
      resolutionNote: note,
    }));
    await act("/findings/resolve-batch", { items, ...versioned("译员") });
  } catch (error) {
    showError(error);
  }
});

$("#compareRuns").addEventListener("click", async () => {
  try {
    const revisionID = workspace.project.currentRevisionID;
    const runs = await api(`/api/projects/${project.id}/validation-runs?revisionID=${encodeURIComponent(revisionID)}`);
    if (runs.length < 2) throw new Error("当前修订还没有两次校验运行");
    const before = runs[runs.length - 2].id;
    const after = runs[runs.length - 1].id;
    const result = await api(`/api/projects/${project.id}/validation-runs/compare?before=${encodeURIComponent(before)}&after=${encodeURIComponent(after)}`);
    $("#runDiff").textContent = JSON.stringify(result, null, 2);
  } catch (error) {
    showError(error);
  }
});

$("#sendReview").addEventListener("click", async () => {
  try {
    await act("/submit-review", versioned("译员"));
  } catch (error) {
    showError(error);
  }
});

$("#reviewDetail").addEventListener("click", async () => {
  try {
    const detail = await api(`/api/projects/${project.id}/review-detail`);
    $("#reviewContext").textContent = JSON.stringify(detail, null, 2);
  } catch (error) {
    showError(error);
  }
});

async function decide(decision) {
  try {
    const reviewer = $("#reviewer").value;
    await act("/review", {
      decision,
      reason: $("#returnReason").value,
      reviewer,
      ...versioned(reviewer),
    });
    $("#reviewDetail").click();
  } catch (error) {
    showError(error);
  }
}

$("#approve").addEventListener("click", () => decide("approve"));
$("#returnReview").addEventListener("click", () => decide("return"));

$("#previewFreeze").addEventListener("click", async () => {
  try {
    const manifest = await api(`/api/projects/${project.id}/freeze/preview`);
    $("#freezePreview").textContent = JSON.stringify(manifest, null, 2);
  } catch (error) {
    showError(error);
  }
});

$("#freeze").addEventListener("click", async () => {
  try {
    await act("/freeze", versioned("负责人"));
    showDownloads();
  } catch (error) {
    showError(error);
  }
});

$("#verify").addEventListener("click", async () => {
  try {
    const result = await api(`/api/projects/${project.id}/verify`);
    $("#verification").textContent = JSON.stringify(result, null, 2);
  } catch (error) {
    showError(error);
  }
});

loadProjects().catch(showError);
