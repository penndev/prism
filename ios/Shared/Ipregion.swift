import Foundation

let ipregionDB = "ipregion.db"

func ipregionFile() -> URL {
    AppGroup.container.appendingPathComponent(ipregionDB)
}

struct IpregionStatus {
    var exists = false
    var size: Int64 = 0
    var version = ""
    var areas = 0
}

/// 文件模式打开：段表解压到本进程临时目录。主 App 和扩展各打开一份。打开失败返回 false。
func openIpregionDb() -> Bool {
    let file = ipregionFile()
    guard FileManager.default.fileExists(atPath: file.path) else { return false }
    var err: NSError?
    return EngineSetIpregionDB(file.path, NSTemporaryDirectory(), &err)
}

func ipregionStatus() -> IpregionStatus {
    guard let st = EngineIpregionStatus() else { return IpregionStatus() }
    let attrs = try? FileManager.default.attributesOfItem(atPath: ipregionFile().path)
    let size = (attrs?[.size] as? NSNumber)?.int64Value
    return IpregionStatus(exists: size != nil, size: size ?? 0, version: st.version, areas: Int(st.areas))
}

func loadAreaTree() -> [AreaUi] {
    guard let data = EngineAreaTree().data(using: .utf8),
          let arr = try? JSONSerialization.jsonObject(with: data) as? [[String: Any]]
    else { return [] }
    func build(_ arr: [[String: Any]]) -> [AreaUi] {
        arr.map { obj in
            AreaUi(
                id: (obj["id"] as? NSNumber)?.int64Value ?? 0,
                parentId: (obj["parent_id"] as? NSNumber)?.int64Value ?? 0,
                name: obj["name"] as? String ?? "",
                children: build(obj["children"] as? [[String: Any]] ?? []),
            )
        }
    }
    return build(arr)
}

func ipregionLookupIds(_ address: String) -> [Int64] {
    guard let list = EngineLookup(address) else { return [] }
    return (0..<list.len()).map { list.get($0) }
}

func installIpregionDb(src: URL, dest: URL) throws {
    if src.standardizedFileURL != dest.standardizedFileURL {
        if FileManager.default.fileExists(atPath: dest.path) {
            try FileManager.default.removeItem(at: dest)
        }
        try FileManager.default.copyItem(at: src, to: dest)
    }
    var err: NSError?
    if !EngineSetIpregionDB(dest.path, NSTemporaryDirectory(), &err) {
        throw err ?? EngineError("open ipregion.db failed")
    }
}

func downloadIpregionDb(url: String, dest: URL, tmp: URL) throws {
    try? FileManager.default.removeItem(at: tmp)
    try httpDownload(url, dest: tmp)
    try installIpregionDb(src: tmp, dest: dest)
    try? FileManager.default.removeItem(at: tmp)
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
