const state = {
  groups: [],
  roles: [],
  employees: [],
  specialties: [],
  posts: [],
  rules: [],
  constraints: [],
  nightShifts: []
};

async function api(path, options = {}) {
  const res = await fetch(`/api${path}`, options);
  if (!res.ok) {
    let msg = "请求失败";
    try {
      const j = await res.json();
      msg = j.error || msg;
    } catch (_) {}
    throw new Error(msg);
  }
  const ct = res.headers.get("content-type") || "";
  if (ct.includes("application/json")) return res.json();
  return res.blob();
}

function selectedGroupId() {
  return Number(document.getElementById("groupSelect").value || 0);
}

function ensureGroupSelected() {
  const gid = selectedGroupId();
  if (!gid) {
    throw new Error("请先新增并选择一个小组");
  }
  return gid;
}

function selectedGroupLabel() {
  const group = state.groups.find((item) => item.id === selectedGroupId());
  return group ? `${group.department} - ${group.name}` : "未选择";
}

function currentNightMonth() {
  return document.getElementById("monthNight").value.trim();
}

function currentConstraintMonth() {
  return document.getElementById("monthConstraint").value.trim();
}

function currentScheduleMonth() {
  return document.getElementById("monthSchedule").value.trim();
}

function renderEmptyRow(colspan, text) {
  return `<tr><td colspan="${colspan}" class="table-empty">${text}</td></tr>`;
}

function setupTabs() {
  const tabs = Array.from(document.querySelectorAll(".tabbar__tab"));
  const panels = Array.from(document.querySelectorAll(".tab-panel"));
  tabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      tabs.forEach((item) => item.classList.toggle("is-active", item === tab));
      const target = tab.dataset.tabTarget;
      panels.forEach((panel) => panel.classList.toggle("is-active", panel.id === target));
    });
  });
}

function setupConfigTabs() {
  const tabs = Array.from(document.querySelectorAll(".subtabbar__tab"));
  const panels = Array.from(document.querySelectorAll(".config-panel"));
  tabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      const target = tab.dataset.configTab;
      tabs.forEach((item) => item.classList.toggle("is-active", item === tab));
      panels.forEach((panel) => panel.classList.toggle("is-active", panel.id === target));
    });
  });
}

function populateSelect(selectId, options, placeholder, valueKey = "value", labelKey = "label") {
  const select = document.getElementById(selectId);
  const isMultiple = select.multiple;
  const previousValues = isMultiple
    ? Array.from(select.selectedOptions).map((item) => item.value)
    : [select.value];
  const base = isMultiple ? "" : `<option value="">${placeholder}</option>`;
  select.innerHTML = base + options.map((item) => `<option value="${item[valueKey]}">${item[labelKey]}</option>`).join("");
  if (isMultiple) {
    Array.from(select.options).forEach((opt) => {
      opt.selected = previousValues.includes(opt.value);
    });
  } else if (options.some((item) => String(item[valueKey]) === previousValues[0])) {
    select.value = previousValues[0];
  }
}

function refreshEmployeeRelatedSelects() {
  populateSelect(
    "employeeRole",
    state.roles.map((item) => ({ value: item.name, label: item.name })),
    "请选择角色"
  );
  populateSelect(
    "employeeCategory",
    state.specialties.map((item) => ({ value: item.name, label: item.name })),
    "未设置专业方向"
  );
  populateSelect(
    "rulePostName",
    state.posts.map((item) => ({ value: item.name, label: item.name })),
    "请选择岗位"
  );
  populateSelect(
    "ruleEmployeeId",
    state.employees.map((item) => ({ value: item.id, label: item.name })),
    "无指定人员"
  );
}

function updateRuleTypeFields() {
  const ruleType = document.getElementById("ruleType").value;
  document.getElementById("ruleDayField").style.display = ruleType === "date" ? "" : "none";
  document.getElementById("ruleWeekdayField").style.display = ruleType === "weekday" ? "" : "none";
}

function updateRuleEmployeeMode() {
  const hasEmployee = Array.from(document.getElementById("ruleEmployeeId").selectedOptions).length > 0;
  document.getElementById("ruleRequiredField").style.display = hasEmployee ? "none" : "";
}

function renderConstraintRoleInputs(existingByRole = {}) {
  const container = document.getElementById("constraintDynamicInputs");
  if (!container) return;
  container.innerHTML = state.roles.map((role) => {
    const value = existingByRole[role.name] ?? 5;
    return `
      <label class="field field--narrow">
        <span class="field__label">${role.name}休息目标</span>
        <input type="number" min="0" value="${value}" data-role-name="${role.name}" class="constraint-role-input" />
      </label>`;
  }).join("");
}

function syncMonths(sourceId) {
  const value = document.getElementById(sourceId).value.trim();
  if (!value) return;
  if (sourceId !== "monthNight") document.getElementById("monthNight").value = value;
  if (sourceId !== "monthConstraint") document.getElementById("monthConstraint").value = value;
  if (sourceId !== "monthSchedule") document.getElementById("monthSchedule").value = value;
}

async function init() {
  setupTabs();
  setupConfigTabs();
  document.getElementById("groupSelect").addEventListener("change", refreshBaseData);
  document.getElementById("ruleType").addEventListener("change", updateRuleTypeFields);
  document.getElementById("ruleEmployeeId").addEventListener("change", updateRuleEmployeeMode);
  document.getElementById("monthNight").addEventListener("change", () => {
    syncMonths("monthNight");
    loadNightShifts();
    renderSummary();
  });
  document.getElementById("monthConstraint").addEventListener("change", () => {
    syncMonths("monthConstraint");
    loadConstraints();
  });
  document.getElementById("monthSchedule").addEventListener("change", () => {
    syncMonths("monthSchedule");
    loadNightShifts();
    renderSummary();
  });
  updateRuleTypeFields();
  updateRuleEmployeeMode();
  await loadGroups();
}

async function loadGroups() {
  state.groups = await api("/groups");
  const sel = document.getElementById("groupSelect");
  const previous = Number(sel.value || 0);
  sel.innerHTML = state.groups.length
    ? state.groups.map((g) => `<option value="${g.id}">${g.department}-${g.name}</option>`).join("")
    : '<option value="">请先新增小组</option>';
  if (state.groups.length) {
    const stillExists = state.groups.some((g) => g.id === previous);
    sel.value = String(stillExists ? previous : state.groups[0].id);
  }
  await refreshBaseData();
}

async function refreshBaseData() {
  const gid = selectedGroupId();
  if (!gid) {
    state.roles = [];
    state.employees = [];
    state.specialties = [];
    state.posts = [];
    state.rules = [];
    state.constraints = [];
    state.nightShifts = [];
    renderRoles();
    renderEmployees();
    renderSpecialties();
    renderPosts();
    renderRules();
    renderConstraints();
    renderNightShifts();
    refreshEmployeeRelatedSelects();
    renderSummary();
    return;
  }
  await Promise.all([
    loadEmployees(),
    loadRoles(),
    loadSpecialties(),
    loadPosts(),
    loadRules(),
    loadConstraints(),
    loadNightShifts()
  ]);
  renderSummary();
}

async function createGroup() {
  try {
    const name = document.getElementById("groupName").value.trim();
    const department = document.getElementById("deptName").value.trim();
    if (!department || !name) {
      throw new Error("请填写科室名称和小组名称");
    }
    const created = await api("/groups", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        name,
        department
      })
    });
    await loadGroups();
    document.getElementById("groupSelect").value = String(created.id);
    await refreshBaseData();
  } catch (e) {
    alert(e.message);
  }
}

async function createEmployee() {
  try {
    const gid = ensureGroupSelected();
    const role = document.getElementById("employeeRole").value;
    const name = document.getElementById("employeeName").value.trim();
    if (!name) {
      throw new Error("请填写员工姓名");
    }
    if (!role) {
      throw new Error("请先选择角色");
    }
    await api("/employees", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        groupId: gid,
        name,
        role,
        category: document.getElementById("employeeCategory").value,
        canNight: document.getElementById("canNight").checked,
        active: true
      })
    });
    await loadEmployees();
    renderSummary();
  } catch (e) {
    alert(e.message);
  }
}

async function createRole() {
  try {
    const gid = ensureGroupSelected();
    const name = document.getElementById("roleName").value.trim();
    if (!name) throw new Error("请填写角色名称");
    await api("/roles", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({ groupId: gid, name })
    });
    document.getElementById("roleName").value = "";
    await loadRoles();
  } catch (e) {
    alert(e.message);
  }
}

async function deleteRole(id) {
  if (!confirm("确定删除该角色吗？")) return;
  await api(`/roles/${id}`, { method: "DELETE" });
  await loadRoles();
}

async function loadRoles() {
  try {
    const gid = selectedGroupId();
    state.roles = gid ? await api(`/roles?groupId=${gid}`) : [];
    renderRoles();
    refreshEmployeeRelatedSelects();
    renderConstraintRoleInputs();
  } catch (e) {
    alert(e.message);
  }
}

function renderRoles() {
  const tbody = document.querySelector("#roleTable tbody");
  if (!tbody) return;
  if (!state.roles.length) {
    tbody.innerHTML = renderEmptyRow(2, "当前小组暂无角色配置");
    return;
  }
  tbody.innerHTML = state.roles.map((role) => `
    <tr>
      <td>${role.name}</td>
      <td><button type="button" class="btn btn--danger" onclick="deleteRole(${role.id})">删除</button></td>
    </tr>`).join("");
}

async function deleteEmployee(id) {
  if (!confirm("确定删除该员工吗？")) return;
  await api(`/employees/${id}`, { method: "DELETE" });
  await loadEmployees();
  renderSummary();
}

async function loadEmployees() {
  try {
    const gid = selectedGroupId();
    state.employees = gid ? await api(`/employees?groupId=${gid}`) : [];
    renderEmployees();
    refreshEmployeeRelatedSelects();
  } catch (e) {
    alert(e.message);
  }
}

function renderEmployees() {
  const tbody = document.querySelector("#employeeTable tbody");
  if (!state.employees.length) {
    tbody.innerHTML = renderEmptyRow(6, "当前小组暂无员工");
    return;
  }
  tbody.innerHTML = state.employees.map((e) => `
    <tr>
      <td>${e.id}</td>
      <td>${e.name}</td>
      <td>${e.role}</td>
      <td>${e.category || "-"}</td>
      <td>${e.canNight ? "是" : "否"}</td>
      <td><button type="button" class="btn btn--danger" onclick="deleteEmployee(${e.id})">删除</button></td>
    </tr>`).join("");
}

async function createSpecialty() {
  try {
    const gid = ensureGroupSelected();
    const name = document.getElementById("specialtyName").value.trim();
    if (!name) {
      throw new Error("请填写专业方向名称");
    }
    await api("/specialties", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({ groupId: gid, name })
    });
    document.getElementById("specialtyName").value = "";
    await loadSpecialties();
  } catch (e) {
    alert(e.message);
  }
}

async function deleteSpecialty(id) {
  if (!confirm("确定删除该专业方向吗？")) return;
  await api(`/specialties/${id}`, { method: "DELETE" });
  await loadSpecialties();
}

async function loadSpecialties() {
  try {
    const gid = selectedGroupId();
    state.specialties = gid ? await api(`/specialties?groupId=${gid}`) : [];
    renderSpecialties();
    refreshEmployeeRelatedSelects();
  } catch (e) {
    alert(e.message);
  }
}

function renderSpecialties() {
  const tbody = document.querySelector("#specialtyTable tbody");
  if (!tbody) return;
  if (!state.specialties.length) {
    tbody.innerHTML = renderEmptyRow(2, "当前小组暂无专业方向配置");
    return;
  }
  tbody.innerHTML = state.specialties.map((item) => `
    <tr>
      <td>${item.name}</td>
      <td><button type="button" class="btn btn--danger" onclick="deleteSpecialty(${item.id})">删除</button></td>
    </tr>`).join("");
}

async function createPost() {
  try {
    const gid = ensureGroupSelected();
    const postName = document.getElementById("postName").value.trim();
    if (!postName) {
      throw new Error("请填写岗位名称");
    }
    await api("/posts", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        groupId: gid,
        name: postName,
        required: Number(document.getElementById("postRequired").value),
        priority: 100,
        enabled: true
      })
    });
    await loadPosts();
    renderSummary();
  } catch (e) {
    alert(e.message);
  }
}

async function loadPosts() {
  try {
    const gid = selectedGroupId();
    state.posts = gid ? await api(`/posts?groupId=${gid}`) : [];
    renderPosts();
    refreshEmployeeRelatedSelects();
  } catch (e) {
    alert(e.message);
  }
}

function renderPosts() {
  const tbody = document.querySelector("#postTable tbody");
  if (!state.posts.length) {
    tbody.innerHTML = renderEmptyRow(2, "当前小组暂无班种配置");
    return;
  }
  tbody.innerHTML = state.posts.map((post) => `
    <tr>
      <td>${post.name}</td>
      <td>${post.required}</td>
    </tr>`).join("");
}

async function createRule() {
  try {
    const gid = ensureGroupSelected();
    const ruleType = document.getElementById("ruleType").value;
    const postName = document.getElementById("rulePostName").value;
    const employeeIds = Array.from(document.getElementById("ruleEmployeeId").selectedOptions).map((opt) => Number(opt.value)).filter((id) => id > 0);
    const dayOfMonth = Number(document.getElementById("ruleDayOfMonth").value || 0);
    const weekday = Number(document.getElementById("ruleWeekday").value || 0);
    if (!postName) {
      throw new Error("请填写规则岗位");
    }
    if (ruleType === "date" && (dayOfMonth < 1 || dayOfMonth > 31)) {
      throw new Error("按日期规则时，日期必须在 1-31 之间");
    }
    if (ruleType === "weekday" && (weekday < 0 || weekday > 6)) {
      throw new Error("按星期规则时，星期必须在 0-6 之间");
    }
    await api("/rules", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        groupId: gid,
        name: "special-rule",
        ruleType,
        dayOfMonth,
        weekday,
        postName,
        employeeIds,
        required: employeeIds.length ? 1 : Number(document.getElementById("ruleRequired").value),
        enabled: true
      })
    });
    await loadRules();
    renderSummary();
  } catch (e) {
    alert(e.message);
  }
}

async function loadRules() {
  try {
    const gid = selectedGroupId();
    state.rules = gid ? await api(`/rules?groupId=${gid}`) : [];
    renderRules();
  } catch (e) {
    alert(e.message);
  }
}

function renderRules() {
  const tbody = document.querySelector("#ruleTable tbody");
  if (!state.rules.length) {
    tbody.innerHTML = renderEmptyRow(5, "当前小组暂无特殊排班规则");
    return;
  }
  tbody.innerHTML = state.rules.map((rule) => {
    const condition = rule.ruleType === "date"
      ? `每月 ${rule.dayOfMonth} 日`
      : `每周 ${["周日", "周一", "周二", "周三", "周四", "周五", "周六"][rule.weekday] || rule.weekday}`;
    return `
      <tr>
        <td>${rule.ruleType === "date" ? "按日期" : "按星期"}</td>
        <td>${condition}</td>
        <td>${rule.postName}</td>
        <td>${rule.employeeName || "-"}</td>
        <td>${rule.required}</td>
      </tr>`;
  }).join("");
}

async function saveConstraint() {
  try {
    const gid = ensureGroupSelected();
    const month = currentConstraintMonth();
    if (!month) {
      throw new Error("请先填写月份");
    }
    const inputs = Array.from(document.querySelectorAll(".constraint-role-input"));
    const payloads = inputs.map((input) => ({
      role: input.dataset.roleName,
      restDaysGoal: Number(input.value)
    }));
    if (!payloads.length) {
      throw new Error("请先配置角色");
    }
    await Promise.all(payloads.map((item) => api("/constraints", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({ groupId: gid, month, ...item })
    })));
    await loadConstraints();
  } catch (e) {
    alert(e.message);
  }
}

async function loadConstraints() {
  try {
    const gid = selectedGroupId();
    const month = currentConstraintMonth();
    state.constraints = gid ? await api(`/constraints?groupId=${gid}&month=${encodeURIComponent(month)}`) : [];
    renderConstraints();
  } catch (e) {
    alert(e.message);
  }
}

function renderConstraints() {
  const tbody = document.querySelector("#constraintTable tbody");
  document.querySelectorAll(".legacy-constraint-input").forEach((el) => {
    el.closest(".field")?.remove();
  });
  if (!state.constraints.length) {
    tbody.innerHTML = renderEmptyRow(3, "当前月份暂无休息规则");
    renderConstraintRoleInputs({});
    return;
  }
  const byRole = Object.fromEntries(state.constraints.map((item) => [item.role, item.restDaysGoal]));
  renderConstraintRoleInputs(byRole);
  tbody.innerHTML = state.constraints.map((item) => `
    <tr>
      <td>${item.month}</td>
      <td>${item.role}</td>
      <td>${item.restDaysGoal}</td>
    </tr>`).join("");
}

async function importNight() {
  try {
    const month = currentNightMonth();
    if (!month) {
      throw new Error("请先填写夜班月份");
    }
    const fileInput = document.getElementById("nightFile");
    if (!fileInput.files.length) throw new Error("请先选择文件");
    const fd = new FormData();
    fd.append("month", month);
    fd.append("file", fileInput.files[0]);
    await api("/night-shifts/import", { method: "POST", body: fd });
    await loadNightShifts();
    renderSummary();
  } catch (e) {
    alert(e.message);
  }
}

async function loadNightShifts() {
  try {
    const month = currentNightMonth() || currentScheduleMonth();
    state.nightShifts = month ? await api(`/night-shifts?month=${encodeURIComponent(month)}`) : [];
    renderNightShifts();
  } catch (e) {
    alert(e.message);
  }
}

function renderNightShifts() {
  const tbody = document.querySelector("#nightTable tbody");
  if (!state.nightShifts.length) {
    tbody.innerHTML = renderEmptyRow(4, "当前月份暂无夜班导入记录");
    return;
  }
  tbody.innerHTML = state.nightShifts.map((item) => `
    <tr>
      <td>${item.month}</td>
      <td>${item.day}</td>
      <td>${item.staffA}</td>
      <td>${item.staffB}</td>
    </tr>`).join("");
}

function renderSummary() {
  document.getElementById("summaryGroup").textContent = selectedGroupLabel();
  document.getElementById("summaryEmployees").textContent = String(state.employees.length);
  document.getElementById("summaryPosts").textContent = String(state.posts.length);
  document.getElementById("summaryRules").textContent = String(state.rules.length);
  document.getElementById("summaryNight").textContent = String(state.nightShifts.length);
}

function confirmGenerate() {
  const month = currentScheduleMonth();
  if (!month) {
    throw new Error("请先填写排班月份");
  }
  const lines = [
    "请确认生成排班前的基础信息是否正确：",
    `小组：${selectedGroupLabel()}`,
    `月份：${month}`,
    `员工数：${state.employees.length}`,
    `班种数：${state.posts.length}`,
    `特殊规则数：${state.rules.length}`,
    `夜班记录数：${state.nightShifts.length}`,
    "",
    "确认无误后点击“确定”开始生成。"
  ];
  return confirm(lines.join("\n"));
}

async function generateSchedule() {
  try {
    ensureGroupSelected();
    if (!confirmGenerate()) return;
    await api("/schedule/generate", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        groupId: selectedGroupId(),
        month: currentScheduleMonth()
      })
    });
    await loadSchedule();
  } catch (e) {
    alert(e.message);
  }
}

async function loadSchedule() {
  try {
    const gid = selectedGroupId();
    const month = currentScheduleMonth();
    const items = await api(`/schedule?groupId=${gid}&month=${encodeURIComponent(month)}`);
    const tbody = document.querySelector("#scheduleTable tbody");
    tbody.innerHTML = items.length
      ? items.map((i) => `<tr><td>${i.day}</td><td>${i.employee}</td><td>${i.shiftName}</td></tr>`).join("")
      : renderEmptyRow(3, "当前月份暂无排班结果");
  } catch (e) {
    alert(e.message);
  }
}

async function exportSchedule() {
  try {
    const gid = selectedGroupId();
    const month = currentScheduleMonth();
    window.open(`/api/schedule/export?groupId=${gid}&month=${encodeURIComponent(month)}`, "_blank");
  } catch (e) {
    alert(e.message);
  }
}

window.createGroup = createGroup;
window.createRole = createRole;
window.deleteRole = deleteRole;
window.createEmployee = createEmployee;
window.deleteEmployee = deleteEmployee;
window.createSpecialty = createSpecialty;
window.deleteSpecialty = deleteSpecialty;
window.createPost = createPost;
window.importNight = importNight;
window.loadNightShifts = loadNightShifts;
window.saveConstraint = saveConstraint;
window.createRule = createRule;
window.generateSchedule = generateSchedule;
window.loadSchedule = loadSchedule;
window.exportSchedule = exportSchedule;

init();
