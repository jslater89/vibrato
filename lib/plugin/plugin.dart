import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/services.dart';
import 'package:path_provider/path_provider.dart';
import 'package:vibrato/stream/stream.dart';
import 'package:vibrato/vibrato.dart';

const MethodChannel channel = const MethodChannel('vibrato');
class Vibrato {
  static Future<Directory> _cachedAssetDir() async {
    Directory root = await getApplicationSupportDirectory();
    Directory assetCache = await Directory(root.path + Platform.pathSeparator + "asset-cache").create();

    return assetCache;
  }

  static String _cachedName(String name) {
    return name.replaceAll(r"/", "-").replaceAll(r"\", "-");
  }

  static Future<String> _cachedPath(String name) async {
    return (await _cachedAssetDir()).path + Platform.pathSeparator + _cachedName(name);
  }

  // TODO: cache more of it, put it in memory
  static Future<bool> ensureAsset(AssetBundle source, String name) async {
    var assetCache = await _cachedAssetDir();
    var cachedName = _cachedName(name);

    try {
      bool exists = false;
      assetCache.listSync().forEach((element) {
        if(element.path.endsWith(cachedName)) {
          exists = true;
          return;
        }
      });

      if(exists) return true;

      File f = File(await _cachedPath(name));
      var bytes = Uint8List.sublistView(await rootBundle.load(name));
      await f.writeAsBytes(bytes);
      return true;
    }
    catch(e, stackTrace) {
      // TODO
      print("error: $e $stackTrace");
      return false;
    }
  }

  static Future<VibratoStream?> playAsset(AssetBundle source, String name) async {
    var result = await ensureAsset(source, name);
    if(!result) return null;

    return playFile(name, File(await _cachedPath(name)));
  }

  static Future<VibratoStream?> playFile(String name, File file) async {
    try {
      var id = await channel.invokeMethod<String>('playFile', {'name': name, 'file': file.absolute.path});
      if(id == null) return null;
      return VibratoStream(id, name: name);
    }
    catch(err) {
      return null;
    }
  }

  static Future<VibratoStream?> playBuffer(String name, Uint8List buffer, AudioFormat format) async {
    try {
      var id = await channel.invokeMethod<String>('playBuffer', {'name': name, 'buffer': buffer, 'format': format.toChannelString()});
      if(id == null) return null;
      return VibratoStream(id, name: name);
    }
    catch(err) {
      return null;
    }
  }

  static Future<List<VibratoStream>> listStreams() async {
    try {
      var ids = await channel.invokeMapMethod<String, String>('listStreams');
      return ids?.map<String, VibratoStream>((id, name) => MapEntry(id, VibratoStream(id, name: name))).values.toList() ?? [];
    }
    catch(err) {
      return [];
    }
  }
}

class StreamMethods {
  static Future<bool> seekStream(String id, int position) async {
    try {
      await channel.invokeMethod('seekStream', {'id': id, 'position': position});
      return true;
    }
    catch(err) {
      return false;
    }
  }

  static Future<bool> pauseStream(String id) async {
    try {
      await channel.invokeMethod('closeStream', {'id': id});
      return true;
    }
    catch(err) {
      return false;
    }
  }

  static Future<StreamInfo?> getStreamInfo(String id) async {
    try {
      var info = await channel.invokeMapMethod<String, dynamic>('streamInfo', {'id': id});
      if(info == null) return null;
      return StreamInfo.fromMap(info);
    }
    catch(err) {
      return null;
    }
  }

  static Future<bool> closeStream(String id) async {
    try {
      await channel.invokeMethod('closeStream', {'id': id});
      return true;
    }
    catch(err) {
      return false;
    }
  }
}

class StreamInfo {
  String name;
  int position;
  int length;
  int sampleRate;

  StreamInfo.fromMap(Map<String, dynamic> map) :
      name = map["name"]!,
      position = map["position"]!,
      length = map["length"]!,
      sampleRate = map["sampleRate"];
}