import { chunkID, type CodeSymbol, type CodeEdge, type SupportedLang } from "./extract.ts";

interface TSNode {
  type: string;
  startIndex: number;
  endIndex: number;
  childCount: number;
  child(i: number): TSNode | null;
  childForFieldName(name: string): TSNode | null;
}

interface EdgeTarget {
  rawTarget: string;
  targetName: string;
  targetQualifier: string;
  targetModuleHint: string;
  receiverTypeHint: string;
}

interface ImportedBinding {
  modulePath: string;
  exportName: string;
}

interface FileImportIndex {
  named: Map<string, ImportedBinding>;
  default: Map<string, ImportedBinding>;
  namespace: Map<string, string>;
}

function emptyTarget(): EdgeTarget {
  return { rawTarget: "", targetName: "", targetQualifier: "", targetModuleHint: "", receiverTypeHint: "" };
}

function normalizeJSImportPath(raw: string): string {
  return raw.replace(/^["'`]|["'`]$/g, "");
}

function pathBase(modulePath: string): string {
  const p = modulePath.endsWith("/") ? modulePath.slice(0, -1) : modulePath;
  const i = p.lastIndexOf("/");
  return i >= 0 ? p.slice(i + 1) : p;
}

function fileExt(docPath: string): string {
  const i = docPath.lastIndexOf(".");
  return i >= 0 ? docPath.slice(i).toLowerCase() : "";
}

function extractFileImports(docPath: string, text: string): FileImportIndex {
  const idx: FileImportIndex = {
    named: new Map(),
    default: new Map(),
    namespace: new Map(),
  };
  const ext = fileExt(docPath);

  if (ext === ".ts" || ext === ".tsx" || ext === ".js" || ext === ".jsx") {
    const combinedRE = /^\s*import\s+([A-Za-z_$][\w$]*)\s*,\s*\{([^}]*)\}\s*from\s*["']([^"']+)["']/gm;
    let m: RegExpExecArray | null;
    while ((m = combinedRE.exec(text)) !== null) {
      idx.default.set(m[1], { modulePath: normalizeJSImportPath(m[3]), exportName: "default" });
      for (const raw of m[2].split(",")) {
        const part = raw.trim();
        if (!part) continue;
        let exportName = part;
        let localName = part;
        const asIdx = part.indexOf(" as ");
        if (asIdx >= 0) {
          exportName = part.slice(0, asIdx).trim();
          localName = part.slice(asIdx + 4).trim();
        }
        idx.named.set(localName, { modulePath: normalizeJSImportPath(m[3]), exportName });
      }
    }

    const namespaceRE = /^\s*import\s+\*\s+as\s+([A-Za-z_$][\w$]*)\s+from\s*["']([^"']+)["']/gm;
    while ((m = namespaceRE.exec(text)) !== null) {
      idx.namespace.set(m[1], normalizeJSImportPath(m[2]));
    }

    const namedRE = /^\s*import\s*\{([^}]*)\}\s*from\s*["']([^"']+)["']/gm;
    while ((m = namedRE.exec(text)) !== null) {
      for (const raw of m[1].split(",")) {
        const part = raw.trim();
        if (!part) continue;
        let exportName = part;
        let localName = part;
        const asIdx = part.indexOf(" as ");
        if (asIdx >= 0) {
          exportName = part.slice(0, asIdx).trim();
          localName = part.slice(asIdx + 4).trim();
        }
        idx.named.set(localName, { modulePath: normalizeJSImportPath(m[2]), exportName });
      }
    }

    const defaultRE = /^\s*import\s+([A-Za-z_$][\w$]*)\s+from\s*["']([^"']+)["']/gm;
    while ((m = defaultRE.exec(text)) !== null) {
      idx.default.set(m[1], { modulePath: normalizeJSImportPath(m[2]), exportName: "default" });
    }
  } else if (ext === ".go") {
    const goImportRE = /^\s*(?:([A-Za-z_][\w]*)\s+)?"([^"]+)"/gm;
    let m: RegExpExecArray | null;
    while ((m = goImportRE.exec(text)) !== null) {
      const modulePath = (m[2] ?? "").trim();
      let alias = (m[1] ?? "").trim();
      if (!modulePath) continue;
      if (!alias) alias = pathBase(modulePath);
      if (!alias || alias === "." || alias === "_") continue;
      idx.namespace.set(alias, modulePath);
    }
  }

  return idx;
}

function lookupBinding(idx: FileImportIndex, local: string): ImportedBinding | null {
  return idx.named.get(local) ?? idx.default.get(local) ?? null;
}

class SymbolVisitor {
  docPath: string;
  language: SupportedLang;
  source: string;
  symbols: CodeSymbol[] = [];
  edges: CodeEdge[] = [];
  imports: FileImportIndex;
  funcStack: string[] = [];
  classStack: string[] = [];

  constructor(docPath: string, language: SupportedLang, source: string) {
    this.docPath = docPath;
    this.language = language;
    this.source = source;
    this.imports = extractFileImports(docPath, source);
  }

  private currentReceiverType(): string {
    return this.classStack.length ? this.classStack[this.classStack.length - 1] : "";
  }

  private nodeText(node: TSNode | null): string {
    if (!node) return "";
    const { startIndex: s, endIndex: e } = node;
    if (s < 0 || e > this.source.length || s >= e) return "";
    return this.source.slice(s, e).trim();
  }

  private addFileSymbol(): void {
    this.symbols.push({
      name: "",
      kind: "file",
      docPath: this.docPath,
      content: "",
      signature: "",
      source: this.source.trim(),
    });
  }

  private addSymbol(name: string, kind: string, node: TSNode): void {
    this.symbols.push({
      name,
      kind,
      docPath: this.docPath,
      content: this.nodeText(node),
      signature: this.extractSignature(node),
      source: this.nodeText(node),
    });
    this.addContainsEdge(name);
    this.addOwnerEdge(name, kind);
  }

  private addContainsEdge(name: string): void {
    if (!name) return;
    this.edges.push({
      from: chunkID(this.docPath, ""),
      to: chunkID(this.docPath, name),
      type: "contains",
      fromPath: this.docPath,
      toPath: this.docPath,
      rawTarget: "",
      targetName: "",
      targetQualifier: "",
      targetModuleHint: "",
      receiverTypeHint: "",
      resolutionStatus: "",
      resolutionConfidence: "",
      resolvedTo: "",
    });
  }

  private addOwnerEdge(name: string, kind: string): void {
    if (!name || kind !== "method" || this.classStack.length === 0) return;
    const owner = this.currentReceiverType();
    if (!owner) return;
    this.edges.push({
      from: chunkID(this.docPath, owner),
      to: chunkID(this.docPath, name),
      type: "has_method",
      fromPath: this.docPath,
      toPath: this.docPath,
      rawTarget: "",
      targetName: "",
      targetQualifier: "",
      targetModuleHint: "",
      receiverTypeHint: "",
      resolutionStatus: "resolved_internal",
      resolutionConfidence: "high",
      resolvedTo: chunkID(this.docPath, name),
    });
  }

  private qualifyEdgeTarget(target: EdgeTarget): EdgeTarget {
    if (!target.targetName) return target;
    if (!target.targetQualifier) {
      const binding = lookupBinding(this.imports, target.targetName);
      if (binding) {
        target.targetModuleHint = binding.modulePath;
        if (binding.exportName && binding.exportName !== "default") {
          target.targetName = binding.exportName;
        }
      }
      return target;
    }
    const modulePath = this.imports.namespace.get(target.targetQualifier);
    if (modulePath) target.targetModuleHint = modulePath;
    return target;
  }

  private addEdge(fromFunc: string, target: EdgeTarget, edgeType: string): void {
    if (!target.targetName) return;
    target = this.qualifyEdgeTarget(target);
    const fromID = fromFunc.trim() ? chunkID(this.docPath, fromFunc) : chunkID(this.docPath, "");
    const resolvedTo = target.targetModuleHint ? `${target.targetModuleHint}::${target.targetName}` : target.targetName;
    this.edges.push({
      from: fromID,
      to: target.targetName,
      type: edgeType,
      fromPath: this.docPath,
      toPath: "",
      rawTarget: target.rawTarget,
      targetName: target.targetName,
      targetQualifier: target.targetQualifier,
      targetModuleHint: target.targetModuleHint,
      receiverTypeHint: target.receiverTypeHint,
      resolutionStatus: "unresolved",
      resolutionConfidence: "low",
      resolvedTo,
    });
  }

  private addFileImportEdges(): void {
    const seen = new Set<string>();
    const add = (modulePath: string) => {
      const mp = modulePath.trim();
      if (!mp || seen.has(mp)) return;
      seen.add(mp);
      this.addEdge("", {
        rawTarget: mp,
        targetName: pathBase(mp),
        targetQualifier: "",
        targetModuleHint: mp,
        receiverTypeHint: "",
      }, "imports");
    };
    for (const b of this.imports.named.values()) add(b.modulePath);
    for (const b of this.imports.default.values()) add(b.modulePath);
    for (const mp of this.imports.namespace.values()) add(mp);
  }

  private extractSignature(node: TSNode): string {
    const parts: string[] = [];
    for (let i = 0; i < node.childCount; i++) {
      const child = node.child(i);
      if (!child) continue;
      const k = child.type;
      if (k === "identifier" || k === "field_identifier" || k === "type_identifier") {
        parts.push(this.source.slice(child.startIndex, child.endIndex));
      } else if (k === "parameter_list" || k === "typed_parameter_list") {
        parts.push(this.source.slice(child.startIndex, child.endIndex));
      }
    }
    return parts.join("").trim();
  }

  private firstChildByKinds(node: TSNode, kinds: string[]): TSNode | null {
    for (let i = 0; i < node.childCount; i++) {
      const c = node.child(i);
      if (!c) continue;
      if (kinds.includes(c.type)) return c;
      const nested = this.firstChildByKinds(c, kinds);
      if (nested) return nested;
    }
    return null;
  }

  private functionName(node: TSNode): string {
    for (let i = 0; i < node.childCount; i++) {
      const c = node.child(i);
      if (c && c.type === "identifier") return this.nodeText(c);
    }
    return this.nodeText(node.childForFieldName("name"));
  }

  private methodName(node: TSNode): string {
    for (let i = 0; i < node.childCount; i++) {
      const c = node.child(i);
      if (!c) continue;
      if (
        c.type === "field_identifier" ||
        c.type === "identifier" ||
        c.type === "property_identifier" ||
        c.type === "private_property_identifier"
      ) {
        return this.nodeText(c);
      }
    }
    return this.nodeText(
      this.firstChildByKinds(node, [
        "property_identifier",
        "private_property_identifier",
        "field_identifier",
        "identifier",
      ]),
    );
  }

  private typeIdentName(node: TSNode): string {
    for (let i = 0; i < node.childCount; i++) {
      const c = node.child(i);
      if (c && (c.type === "type_identifier" || c.type === "identifier")) {
        return this.nodeText(c);
      }
    }
    return "";
  }

  private typeDeclarationSymbol(node: TSNode): { name: string; kind: string } | null {
    const name = this.typeIdentName(node);
    if (!name) return null;
    for (let i = 0; i < node.childCount; i++) {
      const c = node.child(i);
      if (!c) continue;
      switch (c.type) {
        case "struct_type":
        case "class_declaration":
        case "class_definition":
          return { name, kind: "class" };
        case "interface_type":
        case "interface_declaration":
        case "interface_specifier":
          return { name, kind: "interface" };
        case "type_spec": {
          const nested = this.typeDeclarationSymbol(c);
          if (nested) return nested;
        }
      }
    }
    return null;
  }

  private functionVariableName(node: TSNode): string {
    if (node.type !== "variable_declarator") return "";
    let value = node.childForFieldName("value");
    if (!value) value = this.firstChildByKinds(node, ["arrow_function", "function_expression"]);
    if (!value) return "";
    if (value.type === "arrow_function" || value.type === "function_expression") {
      let name = node.childForFieldName("name");
      if (!name) name = this.firstChildByKinds(node, ["identifier"]);
      if (name && name.type === "identifier") return this.nodeText(name);
    }
    return "";
  }

  private functionVariableNames(node: TSNode): string[] {
    if (node.type !== "lexical_declaration" && node.type !== "variable_declaration") return [];
    const names: string[] = [];
    for (let i = 0; i < node.childCount; i++) {
      const c = node.child(i);
      if (!c || c.type !== "variable_declarator") continue;
      const n = this.functionVariableName(c);
      if (n) names.push(n);
    }
    return names;
  }

  private fieldExpressionParts(node: TSNode | null): { qualifier: string; name: string } {
    if (!node) return { qualifier: "", name: "" };
    let qualifier = "";
    let name = "";
    for (let i = 0; i < node.childCount; i++) {
      const c = node.child(i);
      if (!c) continue;
      switch (c.type) {
        case "identifier":
        case "type_identifier":
        case "field_identifier": {
          const text = this.nodeText(c);
          if (!qualifier) qualifier = text;
          else name = text;
          break;
        }
        case "field_expression":
        case "selector_expression":
        case "member_expression": {
          const { qualifier: q, name: n } = this.fieldExpressionParts(c);
          if (n) {
            if (!qualifier) qualifier = n;
            else if (!name) name = n;
          }
          if (!qualifier && q) qualifier = q;
          break;
        }
      }
    }
    if (!name) {
      name = qualifier;
      qualifier = "";
    }
    return { qualifier, name };
  }

  private typeExpressionParts(node: TSNode | null): { qualifier: string; name: string } {
    if (!node) return { qualifier: "", name: "" };
    switch (node.type) {
      case "type_identifier":
      case "identifier":
      case "field_identifier":
        return { qualifier: "", name: this.nodeText(node) };
      case "qualified_type":
      case "selector_expression":
      case "member_expression":
      case "field_expression":
      case "generic_type":
      case "type_instantiation":
      case "new_expression":
      case "composite_literal":
        for (let i = node.childCount - 1; i >= 0; i--) {
          const c = node.child(i);
          if (!c) continue;
          const r = this.typeExpressionParts(c);
          if (r.name) return r;
        }
    }
    return { qualifier: "", name: "" };
  }

  private callExpressionTarget(node: TSNode): EdgeTarget {
    for (let i = 0; i < node.childCount; i++) {
      const c = node.child(i);
      if (!c) continue;
      switch (c.type) {
        case "identifier": {
          const name = this.nodeText(c);
          return { ...emptyTarget(), rawTarget: name, targetName: name };
        }
        case "field_expression":
        case "selector_expression":
        case "member_expression": {
          const { qualifier, name } = this.fieldExpressionParts(c);
          if (name) {
            const raw = qualifier ? `${qualifier}.${name}` : name;
            return { ...emptyTarget(), rawTarget: raw, targetName: name, targetQualifier: qualifier };
          }
        }
      }
    }
    return emptyTarget();
  }

  private instantiatedTarget(node: TSNode): EdgeTarget {
    for (let i = 0; i < node.childCount; i++) {
      const c = node.child(i);
      if (!c) continue;
      const { qualifier, name } = this.typeExpressionParts(c);
      if (name) {
        const raw = qualifier ? `${qualifier}.${name}` : name;
        return { ...emptyTarget(), rawTarget: raw, targetName: name, targetQualifier: qualifier };
      }
    }
    return emptyTarget();
  }

  private extendsTarget(node: TSNode): EdgeTarget {
    const cur = this.currentReceiverType();
    for (let i = 0; i < node.childCount; i++) {
      const c = node.child(i);
      if (!c) continue;
      const { qualifier, name } = this.typeExpressionParts(c);
      if (name && name !== cur) {
        const raw = qualifier ? `${qualifier}.${name}` : name;
        return { ...emptyTarget(), rawTarget: raw, targetName: name, targetQualifier: qualifier };
      }
    }
    return emptyTarget();
  }

  private extractImport(node: TSNode): string {
    switch (node.type) {
      case "import_specifier":
      case "import_declaration":
      case "import_clause":
      case "import_statement":
        for (let i = 0; i < node.childCount; i++) {
          const c = node.child(i);
          if (!c) continue;
          if (c.type === "string_literal") return this.nodeText(c).replace(/^"|"$/g, "");
          const nested = this.extractImport(c);
          if (nested) return nested;
        }
    }
    return "";
  }

  private importTarget(node: TSNode): EdgeTarget {
    const modulePath = this.extractImport(node);
    if (!modulePath) return emptyTarget();
    return {
      ...emptyTarget(),
      rawTarget: modulePath,
      targetName: pathBase(modulePath),
      targetModuleHint: modulePath,
    };
  }

  private addImportEdge(fromFunc: string, node: TSNode): void {
    const target = this.importTarget(node);
    if (!target.targetName) return;
    this.edges.push({
      from: chunkID(this.docPath, fromFunc),
      to: target.targetName,
      type: "imports",
      fromPath: this.docPath,
      toPath: "",
      rawTarget: target.rawTarget,
      targetName: target.targetName,
      targetQualifier: target.targetQualifier,
      targetModuleHint: target.targetModuleHint,
      receiverTypeHint: "",
      resolutionStatus: "unresolved",
      resolutionConfidence: "medium",
      resolvedTo: target.targetModuleHint,
    });
  }

  private importAnchor(): string {
    if (this.funcStack.length) return this.funcStack[this.funcStack.length - 1];
    if (this.classStack.length) return this.classStack[this.classStack.length - 1];
    return "";
  }

  visit(node: TSNode): void {
    switch (node.type) {
      case "function_declaration":
      case "function_definition": {
        const name = this.functionName(node);
        if (name) this.addSymbol(name, "function", node);
        break;
      }
      case "method_declaration":
      case "method_definition": {
        const name = this.methodName(node);
        if (name) this.addSymbol(name, "method", node);
        break;
      }
      case "class_declaration":
      case "class_definition": {
        const name = this.typeIdentName(node);
        if (name) this.addSymbol(name, "class", node);
        break;
      }
      case "interface_declaration":
      case "interface_specifier": {
        const name = this.typeIdentName(node);
        if (name) this.addSymbol(name, "interface", node);
        break;
      }
      case "type_declaration":
      case "type_spec": {
        const decl = this.typeDeclarationSymbol(node);
        if (decl) this.addSymbol(decl.name, decl.kind, node);
        break;
      }
      case "lexical_declaration":
      case "variable_declaration": {
        for (const n of this.functionVariableNames(node)) this.addSymbol(n, "function", node);
        break;
      }
      case "variable_declarator": {
        const name = this.functionVariableName(node);
        if (name) this.addSymbol(name, "function", node);
        break;
      }
      case "class_heritage":
      case "extends_clause": {
        if (this.classStack.length) {
          const target = this.extendsTarget(node);
          if (target.targetName) {
            target.receiverTypeHint = this.currentReceiverType();
            this.addEdge(this.currentReceiverType(), target, "extends");
          }
        }
        break;
      }
      case "call_expression": {
        if (this.funcStack.length) {
          const target = this.callExpressionTarget(node);
          if (target.targetName) {
            target.receiverTypeHint = this.currentReceiverType();
            this.addEdge(this.funcStack[this.funcStack.length - 1], target, "calls");
          }
        }
        break;
      }
      case "new_expression":
      case "composite_literal": {
        if (this.funcStack.length) {
          const target = this.instantiatedTarget(node);
          if (target.targetName) {
            target.receiverTypeHint = this.currentReceiverType();
            this.addEdge(this.funcStack[this.funcStack.length - 1], target, "instantiates");
          }
        }
        break;
      }
      case "import_specifier":
      case "import_declaration":
      case "import_clause":
      case "import_statement": {
        const target = this.importTarget(node);
        if (target.targetName) this.addImportEdge(this.importAnchor(), node);
        break;
      }
    }
  }

  walk(node: TSNode | null): void {
    if (!node) return;
    this.visit(node);
    for (let i = 0; i < node.childCount; i++) {
      const c = node.child(i);
      if (!c) continue;
      const kind = c.type;
      let pushedFunc = false;
      let pushedClass = false;
      switch (kind) {
        case "function_declaration":
        case "function_definition": {
          const n = this.functionName(c);
          if (n) {
            this.funcStack.push(n);
            pushedFunc = true;
          }
          break;
        }
        case "method_declaration":
        case "method_definition": {
          const n = this.methodName(c);
          if (n) {
            this.funcStack.push(n);
            pushedFunc = true;
          }
          break;
        }
        case "variable_declarator": {
          const n = this.functionVariableName(c);
          if (n) {
            this.funcStack.push(n);
            pushedFunc = true;
          }
          break;
        }
        case "class_declaration":
        case "class_definition":
        case "interface_declaration":
        case "interface_specifier": {
          const n = this.typeIdentName(c);
          if (n) {
            this.classStack.push(n);
            pushedClass = true;
          }
          break;
        }
      }
      this.walk(c);
      if (pushedFunc && this.funcStack.length) this.funcStack.pop();
      if (pushedClass && this.classStack.length) this.classStack.pop();
    }
  }

  run(root: TSNode): { symbols: CodeSymbol[]; edges: CodeEdge[] } {
    this.addFileSymbol();
    this.walk(root);
    this.addFileImportEdges();
    return { symbols: this.symbols, edges: this.edges };
  }
}

export function extractSymbolsAndEdgesImpl(
  docPath: string,
  language: SupportedLang,
  source: string,
  root: any,
): { symbols: CodeSymbol[]; edges: CodeEdge[] } {
  const v = new SymbolVisitor(docPath, language, source);
  return v.run(root as TSNode);
}
