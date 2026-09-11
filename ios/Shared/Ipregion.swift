import Foundation

let ipregionDB = "ipregion.db"

func ipregionFile() -> URL {
    AppGroup.container.appendingPathComponent(ipregionDB)
}

func openIpregionDb() {
    let file = ipregionFile()
    if FileManager.default.fileExists(atPath: file.path) {
        _ = try? Engine.setIpregionDB(file.path)
    }
}

func loadAreaTree() -> [AreaUi] {
    let raw = Engine.areaTree()
    if raw.isEmpty || raw == "[]" { return [] }
    guard let data = raw.data(using: .utf8),
          let arr = try? JSONSerialization.jsonObject(with: data) as? [[String: Any]]
    else { return [] }
    return parseAreaTree(arr)
}

func installIpregionDb(src: URL, dest: URL) throws {
    if src.standardizedFileURL != dest.standardizedFileURL {
        if FileManager.default.fileExists(atPath: dest.path) {
            try FileManager.default.removeItem(at: dest)
        }
        try FileManager.default.copyItem(at: src, to: dest)
    }
    try Engine.setIpregionDB(dest.path)
}

func downloadIpregionDb(url: String, dest: URL, tmp: URL) throws {
    try? FileManager.default.removeItem(at: tmp)
    try httpDownload(url, dest: tmp)
    try installIpregionDb(src: tmp, dest: dest)
    try? FileManager.default.removeItem(at: tmp)
}

private func parseAreaTree(_ arr: [[String: Any]]) -> [AreaUi] {
    arr.compactMap { parseArea($0) }
}

private func parseArea(_ obj: [String: Any]) -> AreaUi? {
    let kids = obj["children"] as? [[String: Any]] ?? []
    let id: Int64
    if let n = obj["id"] as? Int64 { id = n }
    else if let n = obj["id"] as? Int { id = Int64(n) }
    else if let n = obj["id"] as? NSNumber { id = n.int64Value }
    else { return nil }
    let parent: Int64
    if let n = obj["parent_id"] as? Int64 { parent = n }
    else if let n = obj["parent_id"] as? Int { parent = Int64(n) }
    else if let n = obj["parent_id"] as? NSNumber { parent = n.int64Value }
    else { parent = 0 }
    return AreaUi(
        id: id,
        parentId: parent,
        name: obj["name"] as? String ?? "",
        children: parseAreaTree(kids),
    )
}

private func httpDownload(_ url: String, dest: URL) throws {
    guard let uri = URL(string: url), let scheme = uri.scheme?.lowercased(),
          scheme == "http" || scheme == "https"
    else { throw EngineError("url must be http or https") }
    var request = URLRequest(url: uri, timeoutInterval: 5 * 60)
    request.setValue("Prism-iOS/0.1", forHTTPHeaderField: "User-Agent")
    let sem = DispatchSemaphore(value: 0)
    var result: Result<Void, Error> = .failure(EngineError("download failed"))
    let task = URLSession.shared.dataTask(with: request) { data, response, error in
        defer { sem.signal() }
        if let error {
            result = .failure(error)
            return
        }
        let code = (response as? HTTPURLResponse)?.statusCode ?? 0
        if !(200...299).contains(code) {
            result = .failure(EngineError("HTTP \(code)"))
            return
        }
        guard let data else {
            result = .failure(EngineError("empty body"))
            return
        }
        if data.count > 64 * 1024 * 1024 {
            result = .failure(EngineError("file too large"))
            return
        }
        do {
            try data.write(to: dest, options: .atomic)
            result = .success(())
        } catch {
            result = .failure(error)
        }
    }
    task.resume()
    if sem.wait(timeout: .now() + 5 * 60) == .timedOut {
        task.cancel()
        throw EngineError("download timeout")
    }
    try result.get()
}
