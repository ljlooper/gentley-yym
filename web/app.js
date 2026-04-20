const state = {
  groups: [],
  roles: [],
  employees: [],
  specialties: [],
  posts: [],
  postDailyRequirements: [],
  postWeekdayRequirements: [],
  rules: [],
  constraints: [],
  nightShifts: [],
  restPlans: [],
  restDebts: []
};

let saveToastTimer = null;

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
  return (
    document.getElementById("restConfigMonth")?.value.trim() ||
    document.getElementById("monthConstraint")?.value.trim() ||
    ""
  );
}

function currentScheduleMonth() {
  return document.getElementById("monthSchedule").value.trim();
}

function currentRestPlanMonth() {
  return (
    document.getElementById("restConfigMonth")?.value.trim() ||
    document.getElementById("monthRestPlan")?.value.trim() ||
    ""
  );
}

function monthFromDateValue(value) {
  return /^\d{4}-\d{2}-\d{2}$/.test(value) ? value.slice(0, 7) : "";
}

function dayFromDateValue(value) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return 0;
  return Number(value.slice(8, 10));
}

function currentRuleMonth() {
  return monthFromDateValue(document.getElementById("ruleDate")?.value.trim() || "") ||
    document.getElementById("ruleMonth")?.value.trim() ||
    currentScheduleMonth();
}

function currentPostRuleMonth() {
  return document.getElementById("postWeekdayMonth")?.value.trim() || currentScheduleMonth();
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

function normalizeDatePickerUI() {
  const legacyRuleDayField = document.getElementById("ruleDayField");
  if (legacyRuleDayField) legacyRuleDayField.style.display = "none";
}

function fillCurrentMonthDefaults() {
  const now = new Date();
  const month = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
  ["monthNight", "restConfigMonth", "monthConstraint", "monthSchedule", "monthRestPlan", "postWeekdayMonth", "ruleMonth"].forEach((id) => {
    const input = document.getElementById(id);
    if (input && !input.value) input.value = month;
  });
}

function populateSelect(selectId, options, placeholder, valueKey = "value", labelKey = "label") {
  const select = document.getElementById(selectId);
  if (!select) return;
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
    "postDailyPostName",
    state.posts.map((item) => ({ value: item.name, label: item.name })),
    "请选择班种"
  );
  populateSelect(
    "postWeekdayPostName",
    state.posts.map((item) => ({ value: item.name, label: item.name })),
    "请选择班种"
  );
  populateSelect(
    "postRangePostName",
    state.posts.map((item) => ({ value: item.name, label: item.name })),
    "请选择班种"
  );
  populateSelect(
    "ruleEmployeeId",
    state.employees.map((item) => ({ value: item.id, label: item.name })),
    "无指定人员"
  );
  populateSelect(
    "restPlanEmployee",
    state.employees.map((item) => ({ value: item.id, label: item.name })),
    "请选择员工"
  );
  const restPlanEmployee = document.getElementById("restPlanEmployee");
  if (restPlanEmployee && !restPlanEmployee.value && state.employees.length) {
    restPlanEmployee.value = String(state.employees[0].id);
  }
  renderCreateRolePicker();
  syncRestPlanFormFromSelection();
}

function renderCreateRolePicker() {
  const select = document.getElementById("employeeRole");
  const picker = document.getElementById("employeeRolePicker");
  if (!select || !picker) return;

  const selectedSet = new Set(Array.from(select.selectedOptions).map((opt) => opt.value));
  const options = state.roles.map((item) => item.name);
  picker.innerHTML = `<div class="role-multi role-multi--create">
    <button type="button" class="role-multi-trigger">请选择角色</button>
    <div class="role-multi-menu">
      ${options.map((name) => `
        <label class="role-multi-option">
          <input type="checkbox" class="employee-role-create-edit" value="${name}" ${selectedSet.has(name) ? "checked" : ""} />
          <span>${name}</span>
        </label>
      `).join("")}
    </div>
  </div>`;

  const syncHiddenSelect = () => {
    let checkedValues = Array.from(picker.querySelectorAll(".employee-role-create-edit"))
      .filter((input) => input.checked)
      .map((input) => input.value);
    if (!checkedValues.length && select.options.length) {
      select.options[0].selected = true;
      checkedValues = [select.options[0].value];
      const first = picker.querySelector(`.employee-role-create-edit[value="${select.options[0].value}"]`);
      if (first) first.checked = true;
    }
    Array.from(select.options).forEach((opt) => {
      opt.selected = checkedValues.includes(opt.value);
    });
    const trigger = picker.querySelector(".role-multi-trigger");
    const text = checkedValues.length ? checkedValues.join("、") : "请选择角色";
    if (trigger) trigger.textContent = text;
  };

  const container = picker.querySelector(".role-multi");
  if (container) {
    bindRoleMultiSelect(container, syncHiddenSelect);
  }

  syncHiddenSelect();
}

function bindRoleMultiSelect(container, onChanged) {
  const trigger = container.querySelector(".role-multi-trigger");
  const menu = container.querySelector(".role-multi-menu");
  if (!trigger || !menu) return;
  trigger.addEventListener("click", (event) => {
    event.stopPropagation();
    document.querySelectorAll(".role-multi.is-open").forEach((el) => {
      if (el !== container) el.classList.remove("is-open");
    });
    container.classList.toggle("is-open");
  });
  menu.querySelectorAll("input[type=\"checkbox\"]").forEach((input) => {
    input.addEventListener("change", () => onChanged(container));
  });
}

function getRoleMultiValues(container) {
  return Array.from(container.querySelectorAll("input[type=\"checkbox\"]"))
    .filter((input) => input.checked)
    .map((input) => input.value);
}

document.addEventListener("click", () => {
  document.querySelectorAll(".role-multi.is-open").forEach((el) => el.classList.remove("is-open"));
});

function updateRuleTypeFields() {
  const typeSelect = document.getElementById("ruleType");
  if (!typeSelect) return;
  const ruleType = typeSelect.value;
  const legacyDayField = document.getElementById("ruleDayField");
  if (legacyDayField) legacyDayField.style.display = "none";
  const dateField = document.getElementById("ruleDateField");
  if (dateField) dateField.style.display = ruleType === "date" ? "" : "none";
  const monthField = document.getElementById("ruleMonthField");
  if (monthField) monthField.style.display = ruleType === "weekday" ? "" : "none";
  const weekdayField = document.getElementById("ruleWeekdayField");
  if (weekdayField) weekdayField.style.display = ruleType === "weekday" ? "" : "none";
}

function updateRuleEmployeeMode() {
  const employeeSelect = document.getElementById("ruleEmployeeId");
  const requiredField = document.getElementById("ruleRequiredField");
  if (!employeeSelect || !requiredField) return;
  const hasEmployee = Array.from(employeeSelect.selectedOptions).length > 0;
  requiredField.style.display = hasEmployee ? "none" : "";
}

function renderConstraintRoleInputs(existingByRole = {}) {
  const container = document.getElementById("constraintDynamicInputs");
  if (!container) return;
  container.innerHTML = state.roles.map((role) => {
    const value = existingByRole[role.name] ?? 5;
    return `
      <label class="field">
        <span class="field__label">${role.name}月度休息目标${role.allowLessRest ? "（允许少休）" : ""}</span>
        <input type="number" min="0" value="${value}" data-role-name="${role.name}" class="constraint-role-input" />
      </label>`;
  }).join("");
}

function syncMonths(sourceId) {
  const source = document.getElementById(sourceId);
  const value = source?.value.trim();
  if (!value) return;
  if (sourceId !== "monthNight" && document.getElementById("monthNight")) document.getElementById("monthNight").value = value;
  if (sourceId !== "restConfigMonth" && document.getElementById("restConfigMonth")) document.getElementById("restConfigMonth").value = value;
  if (sourceId !== "monthConstraint" && document.getElementById("monthConstraint")) document.getElementById("monthConstraint").value = value;
  if (sourceId !== "monthSchedule" && document.getElementById("monthSchedule")) document.getElementById("monthSchedule").value = value;
  if (sourceId !== "monthRestPlan" && document.getElementById("monthRestPlan")) document.getElementById("monthRestPlan").value = value;
  const postWeekdayMonth = document.getElementById("postWeekdayMonth");
  if (postWeekdayMonth && sourceId !== "postWeekdayMonth") postWeekdayMonth.value = value;
  const ruleMonth = document.getElementById("ruleMonth");
  if (ruleMonth && sourceId !== "ruleMonth") ruleMonth.value = value;
}

function ensureToastHost() {
  let host = document.getElementById("saveToast");
  if (host) return host;
  host = document.createElement("div");
  host.id = "saveToast";
  host.className = "save-toast";
  document.body.appendChild(host);
  return host;
}

function showSaveToast(message, type = "success") {
  const host = ensureToastHost();
  host.textContent = message;
  host.classList.remove("is-success", "is-error", "is-show");
  host.classList.add(type === "error" ? "is-error" : "is-success");
  requestAnimationFrame(() => host.classList.add("is-show"));
  if (saveToastTimer) clearTimeout(saveToastTimer);
  saveToastTimer = setTimeout(() => {
    host.classList.remove("is-show");
  }, 1200);
}

async function init() {
  setupTabs();
  setupConfigTabs();
  normalizeDatePickerUI();
  fillCurrentMonthDefaults();
  document.getElementById("groupSelect").addEventListener("change", refreshBaseData);
  document.getElementById("ruleType")?.addEventListener("change", updateRuleTypeFields);
  document.getElementById("ruleEmployeeId")?.addEventListener("change", updateRuleEmployeeMode);
  document.getElementById("ruleMonth")?.addEventListener("change", loadRules);
  document.getElementById("ruleDate")?.addEventListener("change", () => {
    const ruleDate = document.getElementById("ruleDate").value.trim();
    const month = monthFromDateValue(ruleDate);
    if (month) {
      const ruleMonth = document.getElementById("ruleMonth");
      if (ruleMonth) ruleMonth.value = month;
    }
    loadRules();
  });
  document.getElementById("monthNight").addEventListener("change", () => {
    syncMonths("monthNight");
    loadNightShifts();
    renderSummary();
  });
  document.getElementById("monthConstraint")?.addEventListener("change", () => {
    syncMonths("monthConstraint");
    loadConstraints();
  });
  document.getElementById("monthSchedule").addEventListener("change", () => {
    syncMonths("monthSchedule");
    loadNightShifts();
    renderSummary();
  });
  document.getElementById("monthRestPlan")?.addEventListener("change", () => {
    syncMonths("monthRestPlan");
    loadRestPlans();
    loadRestDebts();
  });
  document.getElementById("restConfigMonth")?.addEventListener("change", () => {
    syncMonths("restConfigMonth");
    renderRestPlanFixedDaysPicker();
    loadConstraints();
    loadRestPlans();
    loadRestDebts();
  });
  document.getElementById("restPlanEmployee")?.addEventListener("change", syncRestPlanFormFromSelection);
  document.getElementById("postWeekdayMonth").addEventListener("change", async () => {
    await loadPostDailyRequirements();
    renderPostDailyMatrix();
  });
  updateRuleTypeFields();
  updateRuleEmployeeMode();
  renderRestPlanFixedDaysPicker();
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
    state.postDailyRequirements = [];
    state.postWeekdayRequirements = [];
    state.constraints = [];
    state.nightShifts = [];
    state.restPlans = [];
    state.restDebts = [];
    renderRoles();
    renderEmployees();
    renderSpecialties();
    renderPosts();
    renderRules();
    renderPostDailyRequirements();
    renderPostWeekdayRequirements();
    renderConstraints();
    renderNightShifts();
    renderRestPlans();
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
    loadPostDailyRequirements(),
    loadPostWeekdayRequirements(),
    loadConstraints(),
    loadNightShifts(),
    loadRestPlans(),
    loadRestDebts()
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
    const roleSelect = document.getElementById("employeeRole");
    const roles = Array.from(roleSelect.selectedOptions).map((opt) => opt.value).filter(Boolean);
    const name = document.getElementById("employeeName").value.trim();
    if (!name) {
      throw new Error("请填写员工姓名");
    }
    if (!roles.length) {
      throw new Error("请至少选择一个角色");
    }
    await api("/employees", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        groupId: gid,
        name,
        role: roles[0],
        roles,
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
    const allowLessRest = document.getElementById("roleAllowLessRest").checked;
    if (!name) throw new Error("请填写角色名称");
    await api("/roles", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({ groupId: gid, name, allowLessRest })
    });
    document.getElementById("roleName").value = "";
    document.getElementById("roleAllowLessRest").checked = false;
    await loadRoles();
  } catch (e) {
    alert(e.message);
  }
}

async function updateRoleAllowLessRest(id, allowLessRest) {
  try {
    await api(`/roles/${id}`, {
      method: "PUT",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({ allowLessRest })
    });
    await loadRoles();
  } catch (e) {
    alert(e.message);
  }
}

async function deleteRole(id) {
  if (!confirm("确定删除这个角色吗？")) return;
  await api(`/roles/${id}`, { method: "DELETE" });
  await loadRoles();
}

async function loadRoles() {
  try {
    const gid = selectedGroupId();
    state.roles = gid ? await api(`/roles?groupId=${gid}`) : [];
    renderRoles();
    refreshEmployeeRelatedSelects();
    renderEmployees();
    renderConstraintRoleInputs();
  } catch (e) {
    alert(e.message);
  }
}

function renderRoles() {
  const tbody = document.querySelector("#roleTable tbody");
  if (!tbody) return;
  if (!state.roles.length) {
    tbody.innerHTML = renderEmptyRow(3, "当前小组暂无角色配置");
    return;
  }
  tbody.innerHTML = state.roles.map((role) => `
    <tr>
      <td>${role.name}</td>
      <td>
        <label class="checkbox-wrap">
          <input type="checkbox" ${role.allowLessRest ? "checked" : ""} onchange="updateRoleAllowLessRest(${role.id}, this.checked)" />
          <span>${role.allowLessRest ? "允许少休" : "必须休满"}</span>
        </label>
      </td>
      <td><button type="button" class="btn btn--danger" onclick="deleteRole(${role.id})">删除</button></td>
    </tr>`).join("");
}

async function deleteEmployee(id) {
  if (!confirm("确定删除这位员工吗？")) return;
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
    tbody.innerHTML = renderEmptyRow(5, "当前小组暂无员工");
    return;
  }
  const fallbackRoleSet = new Set();
  state.employees.forEach((e) => {
    const selectedRoles = e.roles && e.roles.length ? e.roles : [e.role];
    selectedRoles.filter(Boolean).forEach((name) => fallbackRoleSet.add(name));
  });
  const roleOptions = state.roles.length
    ? state.roles.map((r) => ({ value: r.name, label: r.name }))
    : Array.from(fallbackRoleSet).map((name) => ({ value: name, label: name }));
  const specialtySet = new Set(state.specialties.map((s) => s.name));
  state.employees.forEach((e) => {
    if (e.category) specialtySet.add(e.category);
  });
  const categoryOptions = [{ value: "", label: "未设置专业方向" }, ...Array.from(specialtySet).map((name) => ({ value: name, label: name }))];
  tbody.innerHTML = state.employees.map((e) => `
    <tr data-employee-id="${e.id}">
      <td>${e.id}</td>
      <td>${e.name}</td>
      <td>
        <div class="employee-identity-cell">
          <div class="employee-role-editor-wrap">
          <div class="role-multi role-multi--row">
            <button type="button" class="role-multi-trigger">${(e.roles && e.roles.length ? e.roles : [e.role]).filter(Boolean).join("、") || "请选择角色"}</button>
            <div class="role-multi-menu">
              ${roleOptions.map((opt) => {
                const selectedRoles = e.roles && e.roles.length ? e.roles : [e.role];
                const checked = selectedRoles.includes(opt.value) ? "checked" : "";
                return `<label class="role-multi-option">
                  <input type="checkbox" class="employee-role-edit" value="${opt.value}" ${checked} />
                  <span>${opt.label}</span>
                </label>`;
              }).join("")}
            </div>
          </div>
          </div>
          <select class="employee-category-edit">
            ${categoryOptions.map((opt) => {
              const selected = (e.category || "") === opt.value ? "selected" : "";
              return `<option value="${opt.value}" ${selected}>${opt.label}</option>`;
            }).join("")}
          </select>
        </div>
      </td>
      <td>
        <label class="checkbox-wrap">
          <input type="checkbox" class="employee-can-night-edit" ${e.canNight ? "checked" : ""} />
          <span>${e.canNight ? "是" : "否"}</span>
        </label>
      </td>
      <td>
        <button type="button" class="btn btn--danger" onclick="deleteEmployee(${e.id})">删除</button>
      </td>
    </tr>`).join("");
  attachEmployeeInlineAutoSave();
}

function attachEmployeeInlineAutoSave() {
  document.querySelectorAll("#employeeTable tbody tr[data-employee-id]").forEach((row) => {
    const roleMulti = row.querySelector(".role-multi");
    const categorySelect = row.querySelector(".employee-category-edit");
    const canNightCheckbox = row.querySelector(".employee-can-night-edit");
    const canNightText = row.querySelector(".checkbox-wrap span");

    if (roleMulti) {
      bindRoleMultiSelect(roleMulti, async (container) => {
        let selectedRoleNames = getRoleMultiValues(container);
        if (selectedRoleNames.length === 0) {
          const first = container.querySelector(".employee-role-edit");
          if (first) first.checked = true;
          selectedRoleNames = getRoleMultiValues(container);
          alert("至少保留一个角色");
        }
        const trigger = container.querySelector(".role-multi-trigger");
        if (trigger) trigger.textContent = selectedRoleNames.join("、");
        await updateEmployeeRow(row);
      });
    }
    if (categorySelect) {
      categorySelect.addEventListener("change", async () => {
        await updateEmployeeRow(row);
      });
    }
    if (canNightCheckbox) {
      canNightCheckbox.addEventListener("change", async () => {
        if (canNightText) canNightText.textContent = canNightCheckbox.checked ? "是" : "否";
        await updateEmployeeRow(row);
      });
    }
  });
}

async function updateEmployeeRow(row) {
  try {
    const id = Number(row.dataset.employeeId || 0);
    if (!id) throw new Error("未找到员工记录");
    const roleMulti = row.querySelector(".role-multi");
    const categorySelect = row.querySelector(".employee-category-edit");
    const canNightCheckbox = row.querySelector(".employee-can-night-edit");
    const roles = roleMulti ? getRoleMultiValues(roleMulti) : [];
    if (!roles.length) throw new Error("请至少选择一个角色");
    await api(`/employees/${id}`, {
      method: "PUT",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        role: roles[0],
        roles,
        category: categorySelect.value || "",
        canNight: Boolean(canNightCheckbox.checked)
      })
    });
    showSaveToast("已保存");
  } catch (e) {
    showSaveToast(e.message || "保存失败", "error");
  }
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
  if (!confirm("确定删除这个专业方向吗？")) return;
  await api(`/specialties/${id}`, { method: "DELETE" });
  await loadSpecialties();
}

async function loadSpecialties() {
  try {
    const gid = selectedGroupId();
    state.specialties = gid ? await api(`/specialties?groupId=${gid}`) : [];
    renderSpecialties();
    refreshEmployeeRelatedSelects();
    renderEmployees();
  } catch (e) {
    alert(e.message);
  }
}

function renderSpecialties() {
  const tbody = document.querySelector("#specialtyTable tbody");
  if (!tbody) return;
  if (!state.specialties.length) {
    tbody.innerHTML = renderEmptyRow(2, "当前小组暂无专业方向");
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
      throw new Error("请填写班种名称");
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
    renderPostDailyRequirements();
    renderPostDailyMatrix();
    refreshEmployeeRelatedSelects();
  } catch (e) {
    alert(e.message);
  }
}

async function loadPostDailyRequirements() {
  try {
    const gid = selectedGroupId();
    const month = currentPostRuleMonth() || currentScheduleMonth() || currentConstraintMonth();
    state.postDailyRequirements = gid && month ? await api(`/post-daily-requirements?groupId=${gid}&month=${encodeURIComponent(month)}`) : [];
    renderPostDailyRequirements();
    renderPostDailyMatrix();
  } catch (e) {
    alert(e.message);
  }
}

async function deletePost(id) {
  if (!confirm("确定删除这个班种吗？")) return;
  await api(`/posts/${id}`, { method: "DELETE" });
  await loadPosts();
  await loadPostDailyRequirements();
  renderSummary();
}

async function loadPostWeekdayRequirements() {
  try {
    const gid = selectedGroupId();
    const month = currentPostRuleMonth() || currentScheduleMonth() || currentConstraintMonth();
    state.postWeekdayRequirements = gid && month ? await api(`/post-weekday-requirements?groupId=${gid}&month=${encodeURIComponent(month)}`) : [];
    renderPostWeekdayRequirements();
    renderPostDemandPreview();
  } catch (e) {
    alert(e.message);
  }
}

function weekdayLabel(value) {
  const map = { 0: "周日", 1: "周一", 2: "周二", 3: "周三", 4: "周四", 5: "周五", 6: "周六" };
  return map[value] || String(value);
}

function daysInMonthValue(month) {
  if (!/^\d{4}-\d{2}$/.test(month)) return 0;
  const [year, monthNum] = month.split("-").map(Number);
  return new Date(year, monthNum, 0).getDate();
}

function parseFixedDaysValue(value) {
  if (!value) return [];
  return value
    .split(",")
    .map((item) => Number(item.trim()))
    .filter((day, index, arr) => Number.isInteger(day) && day > 0 && arr.indexOf(day) === index)
    .sort((a, b) => a - b);
}

function setRestPlanFixedDays(days) {
  const input = document.getElementById("restPlanFixedDays");
  if (input) input.value = days.join(",");
  updateRestPlanFixedDaysSummary(days);
  document.querySelectorAll(".day-picker__day").forEach((button) => {
    const isActive = days.includes(Number(button.dataset.day));
    button.classList.toggle("is-active", isActive);
  });
}

function updateRestPlanFixedDaysSummary(days = parseFixedDaysValue(document.getElementById("restPlanFixedDays")?.value || "")) {
  const summary = document.getElementById("restPlanFixedDaysSummary");
  if (!summary) return;
  summary.textContent = days.length ? `已选择：${days.join("、")} 号` : "请选择固定休息日，可多选。";
}

function renderRestPlanFixedDaysPicker() {
  const picker = document.getElementById("restPlanFixedDaysPicker");
  if (!picker) return;
  const month = currentRestPlanMonth();
  const totalDays = daysInMonthValue(month);
  if (!month || !totalDays) {
    picker.innerHTML = '<div class="day-picker__empty">请选择月份后再设置固定休息日</div>';
    setRestPlanFixedDays([]);
    return;
  }
  const selected = parseFixedDaysValue(document.getElementById("restPlanFixedDays")?.value || "");
  picker.innerHTML = Array.from({ length: totalDays }, (_, index) => {
    const day = index + 1;
    const active = selected.includes(day) ? " is-active" : "";
    const weekday = weekdayLabel(new Date(`${month}-${String(day).padStart(2, "0")}T00:00:00`).getDay());
    return `<button type="button" class="day-picker__day${active}" data-day="${day}">
      <strong>${day}号</strong>
      <span>${weekday}</span>
    </button>`;
  }).join("");
  picker.querySelectorAll(".day-picker__day").forEach((button) => {
    button.addEventListener("click", () => {
      const next = new Set(parseFixedDaysValue(document.getElementById("restPlanFixedDays")?.value || ""));
      const day = Number(button.dataset.day);
      if (next.has(day)) {
        next.delete(day);
      } else {
        next.add(day);
      }
      setRestPlanFixedDays(Array.from(next).sort((a, b) => a - b));
    });
  });
  updateRestPlanFixedDaysSummary(selected);
}

function syncRestPlanFormFromSelection() {
  const employeeId = Number(document.getElementById("restPlanEmployee")?.value || 0);
  const matched = state.restPlans.find((item) => item.employeeId === employeeId);
  const floatInput = document.getElementById("restPlanFloatDays");
  const noteInput = document.getElementById("restPlanNote");
  if (floatInput) floatInput.value = matched ? matched.floatDays : -1;
  if (noteInput) noteInput.value = matched?.note || "";
  setRestPlanFixedDays(parseFixedDaysValue(matched?.fixedDays || ""));
  renderRestPlanFixedDaysPicker();
}

function buildDemandPreviewRows(month) {
  if (!month || !state.posts.length) return [];
  const totalDays = daysInMonthValue(month);
  if (!totalDays) return [];

  const weekdayOverrides = {};
  state.postWeekdayRequirements.forEach((item) => {
    if (item.month !== month) return;
    weekdayOverrides[`${item.weekday}__${item.postName}`] = item.required;
  });

  const dailyOverrides = {};
  state.postDailyRequirements.forEach((item) => {
    if (item.month !== month) return;
    dailyOverrides[`${item.day}__${item.postName}`] = item.required;
  });

  const rows = [];
  for (let day = 1; day <= totalDays; day++) {
    const weekday = weekdayOfDay(month, day);
    const byPost = [];
    let totalRequired = 0;
    state.posts.forEach((post) => {
      const weekdayKey = `${weekday}__${post.name}`;
      const dailyKey = `${day}__${post.name}`;
      const required = dailyOverrides[dailyKey] ?? weekdayOverrides[weekdayKey] ?? post.required ?? 0;
      byPost.push({ name: post.name, required });
      totalRequired += Math.max(0, required);
    });
    rows.push({ day, weekday, byPost, totalRequired });
  }
  return rows;
}

function renderPostDemandPreview() {
  const tbody = document.querySelector("#postDemandPreviewTable tbody");
  const badge = document.getElementById("postPreviewMonthBadge");
  if (!tbody || !badge) return;

  const month = currentPostRuleMonth() || document.getElementById("postWeekdayMonth").value.trim();
  badge.textContent = month || "未选择月份";

  if (!month) {
    tbody.innerHTML = renderEmptyRow(4, "请先选择月份，再查看月度需求预览");
    renderPostWeekdayMatrix();
    return;
  }

  const rows = buildDemandPreviewRows(month);
  if (!rows.length) {
    tbody.innerHTML = renderEmptyRow(4, "请先配置班种和模板");
    return;
  }

  tbody.innerHTML = rows.map((row) => {
    const summary = row.byPost.map((item) => `<span><strong>${item.name}</strong> ${item.required}</span>`).join("、");
    return `<tr>
      <td>${month}-${String(row.day).padStart(2, "0")}</td>
      <td>${weekdayLabel(row.weekday)}</td>
      <td>${summary}</td>
      <td>${row.totalRequired}</td>
    </tr>`;
  }).join("");
}

function renderPostWeekdayRequirements() {
  const tbody = document.querySelector("#postWeekdayTable tbody");
  if (!tbody) return;
  if (!state.postWeekdayRequirements.length) {
    tbody.innerHTML = renderEmptyRow(4, "当前月份暂无星期模板配置");
    renderPostWeekdayMatrix();
    return;
  }
  tbody.innerHTML = state.postWeekdayRequirements.map((item) => `
    <tr>
      <td>${item.month}</td>
      <td>${weekdayLabel(item.weekday)}</td>
      <td>${item.postName}</td>
      <td>${item.required}</td>
    </tr>`).join("");
  renderPostWeekdayMatrix();
}

function renderPostWeekdayMatrix() {
  const tbody = document.querySelector("#postWeekdayMatrixTable tbody");
  if (!tbody) return;
  if (!state.posts.length) {
    tbody.innerHTML = renderEmptyRow(8, "请先配置班种");
    return;
  }
  const byKey = {};
  state.postWeekdayRequirements.forEach((item) => {
    byKey[`${item.postName}__${item.weekday}`] = item.required;
  });
  const weekdayOrder = [1, 2, 3, 4, 5, 6, 0];
  tbody.innerHTML = state.posts.map((post) => {
    return `<tr>
      <td>${post.name}</td>
      ${weekdayOrder.map((weekday) => {
        const key = `${post.name}__${weekday}`;
        const value = byKey[key] ?? post.required ?? 0;
        return `<td><input type="number" min="0" class="weekday-matrix-input" data-post-name="${post.name}" data-weekday="${weekday}" value="${value}" /></td>`;
      }).join("")}
    </tr>`;
  }).join("");
}

function renderPostDailyRequirements() {
  const tbody = document.querySelector("#postDailyTable tbody");
  if (!tbody) return;
  if (!state.postDailyRequirements.length) {
    tbody.innerHTML = renderEmptyRow(4, "当前月份暂无每日覆盖配置");
    return;
  }
  tbody.innerHTML = state.postDailyRequirements.map((item) => `
    <tr>
      <td>${item.month}</td>
      <td>${item.day}</td>
      <td>${item.postName}</td>
      <td>${item.required}</td>
    </tr>`).join("");
}

function renderPostDailyMatrix() {
  const thead = document.querySelector("#postDailyMatrixTable thead");
  const tbody = document.querySelector("#postDailyMatrixTable tbody");
  const badge = document.getElementById("postPreviewMonthBadge");
  if (!thead || !tbody || !badge) return;

  const month = currentPostRuleMonth();
  badge.textContent = month || "未选择月份";
  if (!month) {
    thead.innerHTML = "";
    tbody.innerHTML = renderEmptyRow(1, "请先选择月份");
    return;
  }

  const totalDays = daysInMonthValue(month);
  if (!totalDays || !state.posts.length) {
    thead.innerHTML = "";
    tbody.innerHTML = renderEmptyRow(1, "请先配置班种模板");
    return;
  }

  const dailyOverrides = {};
  state.postDailyRequirements.forEach((item) => {
    if (item.month !== month) return;
    dailyOverrides[`${item.day}__${item.postName}`] = item.required;
  });

  thead.innerHTML = `<tr>
    <th>日期</th>
    ${state.posts.map((post) => `<th>${post.name}</th>`).join("")}
    <th>总人数</th>
  </tr>`;

  tbody.innerHTML = Array.from({ length: totalDays }, (_, index) => {
    const day = index + 1;
    const weekday = weekdayLabel(new Date(`${month}-${String(day).padStart(2, "0")}T00:00:00`).getDay());
    let total = 0;
    const cells = state.posts.map((post) => {
      const value = dailyOverrides[`${day}__${post.name}`] ?? post.required ?? 0;
      total += Number(value || 0);
      return `<td><input type="number" min="0" class="matrix-cell-input" data-post-name="${post.name}" data-day="${day}" value="${value}" /></td>`;
    }).join("");
    return `<tr>
      <td><div class="matrix-day-meta"><strong>${day} 日</strong><span>${weekday}</span></div></td>
      ${cells}
      <td class="matrix-total" data-total-for-day="${day}">${total}</td>
    </tr>`;
  }).join("");

  document.querySelectorAll(".matrix-cell-input").forEach((input) => {
    input.addEventListener("input", updatePostDailyMatrixTotals);
  });
}

function updatePostDailyMatrixTotals() {
  const totals = {};
  document.querySelectorAll(".matrix-cell-input").forEach((input) => {
    const day = input.dataset.day;
    totals[day] = (totals[day] || 0) + Number(input.value || 0);
  });
  Object.entries(totals).forEach(([day, total]) => {
    const target = document.querySelector(`[data-total-for-day="${day}"]`);
    if (target) target.textContent = String(total);
  });
}

async function savePostDailyMatrix() {
  try {
    const gid = ensureGroupSelected();
    const month = currentPostRuleMonth();
    if (!month) throw new Error("Please select a month first");
    const items = Array.from(document.querySelectorAll(".matrix-cell-input")).map((input) => ({
      groupId: gid,
      month,
      day: Number(input.dataset.day),
      postName: input.dataset.postName,
      required: Math.max(0, Number(input.value || 0))
    }));
    if (!items.length) throw new Error("No matrix data to save");
    await api("/post-daily-requirements/bulk", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({ groupId: gid, month, items })
    });
    await loadPostDailyRequirements();
    showSaveToast("Saved monthly demand table");
  } catch (e) {
    alert(e.message);
  }
}

async function savePostDailyRequirement() {
  try {
    const gid = ensureGroupSelected();
    const dateValue = document.getElementById("postDailyDate").value.trim();
    const month = monthFromDateValue(dateValue);
    const day = dayFromDateValue(dateValue);
    if (!month || !day) throw new Error("Please select a date");
    const postName = document.getElementById("postDailyPostName").value;
    const required = Number(document.getElementById("postDailyRequired").value);
    if (!postName) throw new Error("Please select a shift post");
    if (required < 0) throw new Error("Required count cannot be negative");
    await api("/post-daily-requirements", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({ groupId: gid, month, day, postName, required })
    });
    document.getElementById("postWeekdayMonth").value = month;
    await loadPostDailyRequirements();
  } catch (e) {
    alert(e.message);
  }
}

async function savePostWeekdayRequirement() {
  try {
    const gid = ensureGroupSelected();
    const month = document.getElementById("postWeekdayMonth").value.trim() || currentPostRuleMonth();
    if (!month) throw new Error("请先选择月份");
    const weekday = Number(document.getElementById("postWeekday").value);
    const postName = document.getElementById("postWeekdayPostName").value;
    const required = Number(document.getElementById("postWeekdayRequired").value);
    if (!postName) throw new Error("请先选择班种");
    if (weekday < 0 || weekday > 6) throw new Error("星期必须在 0 到 6 之间");
    if (required < 0) throw new Error("人数不能小于 0");
    await api("/post-weekday-requirements", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({ groupId: gid, month, weekday, postName, required })
    });
    document.getElementById("postWeekdayMonth").value = month;
    await loadPostWeekdayRequirements();
  } catch (e) {
    alert(e.message);
  }
}

async function savePostWeekdayMatrix() {
  try {
    const gid = ensureGroupSelected();
    const month = document.getElementById("postWeekdayMonth").value.trim() || currentPostRuleMonth();
    if (!month) throw new Error("请先选择月份");
    const inputs = Array.from(document.querySelectorAll(".weekday-matrix-input"));
    if (!inputs.length) throw new Error("当前没有可保存的星期模板");
    const payloads = inputs.map((input) => ({
      groupId: gid,
      month,
      weekday: Number(input.dataset.weekday),
      postName: input.dataset.postName,
      required: Number(input.value || 0)
    }));
    await Promise.all(payloads.map((item) => api("/post-weekday-requirements", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify(item)
    })));
    document.getElementById("postWeekdayMonth").value = month;
    await loadPostWeekdayRequirements();
    showSaveToast("星期模板整表已保存");
  } catch (e) {
    alert(e.message);
  }
}

function weekdayOfDay(month, day) {
  const date = new Date(`${month}-${String(day).padStart(2, "0")}T00:00:00`);
  return Number.isNaN(date.getTime()) ? -1 : date.getDay();
}

async function applyPostRangeOverride() {
  try {
    const gid = ensureGroupSelected();
    const startValue = document.getElementById("postRangeStartDate").value.trim();
    const endValue = document.getElementById("postRangeEndDate").value.trim();
    const startDate = startValue ? new Date(`${startValue}T00:00:00`) : null;
    const endDate = endValue ? new Date(`${endValue}T00:00:00`) : null;
    const weekdayFilterRaw = document.getElementById("postRangeWeekday").value;
    const weekdayFilter = weekdayFilterRaw === "" ? null : Number(weekdayFilterRaw);
    const postName = document.getElementById("postRangePostName").value;
    const required = Number(document.getElementById("postRangeRequired").value);
    if (!startDate || !endDate || Number.isNaN(startDate.getTime()) || Number.isNaN(endDate.getTime())) throw new Error("Please choose a start and end date");
    if (startDate > endDate) throw new Error("Start date must be before end date");
    if (!postName) throw new Error("Please select a shift post");
    if (required < 0) throw new Error("Required count cannot be negative");
    const tasks = [];
    for (let cursor = new Date(startDate); cursor <= endDate; cursor.setDate(cursor.getDate() + 1)) {
      if (weekdayFilter !== null && cursor.getDay() !== weekdayFilter) continue;
      const month = `${cursor.getFullYear()}-${String(cursor.getMonth() + 1).padStart(2, "0")}`;
      const day = cursor.getDate();
      tasks.push(api("/post-daily-requirements", {
        method: "POST",
        headers: {"Content-Type":"application/json"},
        body: JSON.stringify({ groupId: gid, month, day, postName, required })
      }));
    }
    if (!tasks.length) throw new Error("No matching dates found in the selected range");
    await Promise.all(tasks);
    await loadPostDailyRequirements();
    showSaveToast(`Applied to ${tasks.length} day(s)`);
  } catch (e) {
    alert(e.message);
  }
}

async function copyPostRequirementsFromPrevMonth() {
  try {
    const gid = ensureGroupSelected();
    const month = currentPostRuleMonth();
    if (!month) throw new Error("请先选择月份");
    const ok = confirm("将清空本月已有的配置并复制上月数据，是否继续？");
    if (!ok) return;
    await api("/post-requirements/copy-from-prev-month", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({ groupId: gid, month })
    });
    await Promise.all([loadPostWeekdayRequirements(), loadPostDailyRequirements()]);
  } catch (e) {
    alert(e.message);
  }
}

function renderPosts() {
  const tbody = document.querySelector("#postTable tbody");
  if (!state.posts.length) {
    tbody.innerHTML = renderEmptyRow(3, "当前小组暂无班种配置");
    return;
  }
  tbody.innerHTML = state.posts.map((post) => `
    <tr>
      <td>${post.name}</td>
      <td>${post.required}</td>
      <td><button type="button" class="btn btn--danger" onclick="deletePost(${post.id})">删除</button></td>
    </tr>`).join("");
}

async function createRule() {
  try {
    const gid = ensureGroupSelected();
    const ruleType = document.getElementById("ruleType").value;
    const postName = document.getElementById("rulePostName").value;
    const employeeIds = Array.from(document.getElementById("ruleEmployeeId").selectedOptions).map((opt) => Number(opt.value)).filter((id) => id > 0);
    const dateValue = document.getElementById("ruleDate")?.value.trim() || "";
    const month = ruleType === "date" ? monthFromDateValue(dateValue) : (document.getElementById("ruleMonth")?.value.trim() || "");
    const dayOfMonth = ruleType === "date" ? dayFromDateValue(dateValue) : 0;
    const weekday = Number(document.getElementById("ruleWeekday").value || 0);
    if (!postName) throw new Error("Please choose a target post");
    if (!employeeIds.length) throw new Error("Please choose at least one employee to lock");
    if (!month) throw new Error(ruleType === "date" ? "Please choose a specific date" : "Please choose an effective month");
    if (ruleType === "date" && (dayOfMonth < 1 || dayOfMonth > 31)) throw new Error("Invalid date");
    if (ruleType === "weekday" && (weekday < 0 || weekday > 6)) throw new Error("Weekday must be between 0 and 6");
    await api("/rules", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({
        groupId: gid,
        month,
        name: "special-rule",
        ruleType,
        dayOfMonth,
        weekday,
        postName,
        employeeIds,
        required: 1,
        enabled: true
      })
    });
    const ruleMonth = document.getElementById("ruleMonth");
    if (ruleMonth && month) ruleMonth.value = month;
    await loadRules();
    renderSummary();
  } catch (e) {
    alert(e.message);
  }
}

async function loadRules() {
  try {
    const gid = selectedGroupId();
    const month = currentRuleMonth();
    const query = month ? `/rules?groupId=${gid}&month=${encodeURIComponent(month)}` : `/rules?groupId=${gid}`;
    state.rules = gid ? await api(query) : [];
    renderRules();
  } catch (e) {
    alert(e.message);
  }
}

function renderRules() {
  const tbody = document.querySelector("#ruleTable tbody");
  if (!tbody) return;
  if (!state.rules.length) {
    tbody.innerHTML = renderEmptyRow(5, "当前小组暂无特殊规则");
    return;
  }
  tbody.innerHTML = state.rules.map((rule) => {
    const condition = rule.ruleType === "date"
      ? `${rule.month || "-"} / ${String(rule.dayOfMonth).padStart(2, "0")} 日`
      : `${rule.month || "-"} / ${weekdayLabel(rule.weekday)}`;
    return `
      <tr>
        <td>${rule.ruleType === "date" ? "按日期" : "按星期"}</td>
        <td>${condition}</td>
        <td>${rule.postName}</td>
        <td>${rule.employeeName || "-"}</td>
        <td><button type="button" class="btn btn--danger" onclick="deleteRule(${rule.id})">删除</button></td>
      </tr>`;
  }).join("");
}

async function deleteRule(id) {
  if (!confirm("确定删除这条特殊规则吗？")) return;
  await api(`/rules/${id}`, { method: "DELETE" });
  await loadRules();
  renderSummary();
}

async function saveConstraint() {
  try {
    const gid = ensureGroupSelected();
    const month = currentConstraintMonth();
    if (!month) {
      throw new Error("请先选择月份");
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
  if (!tbody) return;
  document.querySelectorAll(".legacy-constraint-input").forEach((el) => {
    el.closest(".field")?.remove();
  });
  if (!state.constraints.length) {
    tbody.innerHTML = renderEmptyRow(3, "当前月份暂无休息目标");
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
      throw new Error("请先选择夜班月份");
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
  document.getElementById("summaryRules").textContent = String(state.postDailyRequirements.length);
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
    const precheck = await api(`/schedule/precheck?groupId=${selectedGroupId()}&month=${encodeURIComponent(currentScheduleMonth())}`);
    if (!precheck.ok) {
      throw new Error(precheck.message || "棰勬鏌ユ湭閫氳繃");
    }
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
    await loadRemarks();
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

async function loadRestPlans() {
  try {
    const gid = selectedGroupId();
    const month = currentRestPlanMonth() || currentConstraintMonth();
    state.restPlans = gid && month ? await api(`/employee-rest-plans?groupId=${gid}&month=${encodeURIComponent(month)}`) : [];
    renderRestPlans();
    syncRestPlanFormFromSelection();
    renderRestPlanFixedDaysPicker();
  } catch (e) {
    alert(e.message);
  }
}

async function loadRestDebts() {
  try {
    const gid = selectedGroupId();
    const month = currentRestPlanMonth() || currentConstraintMonth();
    state.restDebts = gid ? await api(`/rest-debts?groupId=${gid}&month=${encodeURIComponent(month)}`) : [];
    renderDebtAlert();
  } catch (e) {
    // ignore
  }
}

function renderDebtAlert() {
  const el = document.getElementById("restDebtAlert");
  if (!el) return;
  el.style.display = "none";
  el.innerHTML = "";
}

function renderRestPlans() {
  const tbody = document.querySelector("#restPlanTable tbody");
  if (!tbody) return;
  if (!state.restPlans.length) {
    tbody.innerHTML = renderEmptyRow(5, "当前月份暂无个人休息配置，员工将按角色休息目标自动排班");
    return;
  }
  tbody.innerHTML = state.restPlans.map((p) => {
    const floatLabel = p.floatDays < 0 ? "自动（按角色目标）" : String(p.floatDays);
    return `<tr>
      <td>${p.employeeName}</td>
      <td>${p.fixedDays || "-"}</td>
      <td>${floatLabel}</td>
      <td>${p.note || "-"}</td>
      <td><button type="button" class="btn btn--danger" onclick="deleteRestPlan(${p.id})">删除</button></td>
    </tr>`;
  }).join("");
}

async function saveRestPlan() {
  try {
    const gid = ensureGroupSelected();
    const month = currentRestPlanMonth();
    if (!month) throw new Error("请先选择月份");
    const employeeSelect = document.getElementById("restPlanEmployee");
    if (employeeSelect && !employeeSelect.value && state.employees.length) {
      employeeSelect.value = String(state.employees[0].id);
    }
    const employeeId = Number(employeeSelect?.value || 0);
    if (!employeeId) throw new Error("请先选择员工");
    const fixedDays = document.getElementById("restPlanFixedDays").value.trim();
    const floatDays = Number(document.getElementById("restPlanFloatDays").value);
    const note = document.getElementById("restPlanNote").value.trim();
    await api("/employee-rest-plans", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ groupId: gid, month, employeeId, fixedDays, floatDays, note })
    });
    await loadRestPlans();
  } catch (e) {
    alert(e.message);
  }
}

async function deleteRestPlan(id) {
  if (!confirm("确定删除这条配置吗？")) return;
  await api(`/employee-rest-plans/${id}`, { method: "DELETE" });
  await loadRestPlans();
}

async function loadRemarks() {
  try {
    const gid = selectedGroupId();
    const month = currentScheduleMonth();
    const card = document.getElementById("remarksCard");
    const list = document.getElementById("remarksList");
    if (!gid || !month) { card.style.display = "none"; return; }
    const items = await api(`/schedule/remarks?groupId=${gid}&month=${encodeURIComponent(month)}`);
    if (!items || items.length === 0) { card.style.display = "none"; return; }
    card.style.display = "";
    const tagMeta = {
      debt:       { label: "休息欠账", cls: "remark--debt" },
      crossmonth: { label: "跨月处理", cls: "remark--cross" },
      makeup:     { label: "上月补偿", cls: "remark--makeup" }
    };
    list.innerHTML = items.map(r => {
      const meta = tagMeta[r.tag] || { label: r.tag, cls: "" };
      return `<div class="remark-item ${meta.cls}">
        <span class="remark-tag">${meta.label}</span>
        <span class="remark-content">${r.content}</span>
      </div>`;
    }).join("");
  } catch (e) {
    // ignore
  }
}

window.createGroup = createGroup;
window.createRole = createRole;
window.updateRoleAllowLessRest = updateRoleAllowLessRest;
window.deleteRole = deleteRole;
window.createEmployee = createEmployee;
window.deleteEmployee = deleteEmployee;
window.createSpecialty = createSpecialty;
window.deleteSpecialty = deleteSpecialty;
window.createPost = createPost;
window.deletePost = deletePost;
window.savePostDailyMatrix = savePostDailyMatrix;
window.savePostDailyRequirement = savePostDailyRequirement;
window.savePostWeekdayRequirement = savePostWeekdayRequirement;
window.savePostWeekdayMatrix = savePostWeekdayMatrix;
window.applyPostRangeOverride = applyPostRangeOverride;
window.copyPostRequirementsFromPrevMonth = copyPostRequirementsFromPrevMonth;
window.importNight = importNight;
window.loadNightShifts = loadNightShifts;
window.saveConstraint = saveConstraint;
window.createRule = createRule;
window.deleteRule = deleteRule;
window.generateSchedule = generateSchedule;
window.loadSchedule = loadSchedule;
window.exportSchedule = exportSchedule;
window.saveRestPlan = saveRestPlan;
window.deleteRestPlan = deleteRestPlan;

init();






