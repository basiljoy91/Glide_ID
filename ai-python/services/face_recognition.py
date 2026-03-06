"""
Face Recognition Service using DeepFace
"""

import numpy as np
import os
from PIL import Image
from typing import Tuple, Optional, List, Dict
import cv2
from deepface import DeepFace
import tensorflow as tf

from config import settings


class FaceRecognitionService:
    """Handle face detection and vectorization using DeepFace"""
    
    def __init__(self):
        # ArcFace-only enterprise pipeline (single embedding space).
        self.model_name = "ArcFace"
        self.backend = settings.DEEPFACE_BACKEND
        self.detector = settings.DEEPFACE_DETECTOR
        self.model_priority = [self.model_name]
        self.detector_priority = self._ordered_unique([
            self.detector,
            "retinaface",
            "mtcnn",
            "opencv",
        ])
        
        # Suppress TensorFlow warnings
        tf.get_logger().setLevel('ERROR')
        os.environ['TF_CPP_MIN_LOG_LEVEL'] = '2'

    def warmup(self) -> Dict[str, str]:
        """
        Preload ArcFace model and detector paths to avoid first-request latency spikes.
        """
        status: Dict[str, str] = {}
        try:
            DeepFace.build_model(self.model_name)
            status["model"] = "loaded"
        except Exception as exc:
            status["model"] = f"error: {exc}"

        dummy = np.zeros((224, 224, 3), dtype=np.uint8)
        for detector_backend in self.detector_priority:
            try:
                DeepFace.extract_faces(
                    img_path=dummy,
                    detector_backend=detector_backend,
                    enforce_detection=False,
                    align=True,
                )
                status[f"detector:{detector_backend}"] = "ready"
            except Exception as exc:
                status[f"detector:{detector_backend}"] = f"error: {exc}"
        return status

    def _ordered_unique(self, values: List[str]) -> List[str]:
        out: List[str] = []
        seen = set()
        for v in values:
            if not v:
                continue
            key = str(v).strip()
            if not key or key in seen:
                continue
            seen.add(key)
            out.append(key)
        return out

    def _extract_embedding_from_result(self, result) -> Optional[np.ndarray]:
        if not result:
            return None
        if isinstance(result, list):
            payload = result[0] if result else None
        elif isinstance(result, dict):
            payload = result
        else:
            payload = None
        if not payload or "embedding" not in payload:
            return None
        return np.array(payload["embedding"], dtype=np.float32)

    def _represent_single(self, img_array: np.ndarray, model_name: str, detector_backend: str) -> Optional[np.ndarray]:
        try:
            embedding = DeepFace.represent(
                img_path=img_array,
                model_name=model_name,
                detector_backend=detector_backend,
                enforce_detection=True,
                align=True,
            )
            vec = self._extract_embedding_from_result(embedding)
            if vec is not None:
                return vec
        except Exception as detection_error:
            msg = str(detection_error).lower()
            no_face_signals = [
                "face could not be detected",
                "face cannot be detected",
                "could not detect face",
                "enforce_detection",
            ]
            if not any(signal in msg for signal in no_face_signals):
                return None

        try:
            embedding = DeepFace.represent(
                img_path=img_array,
                model_name=model_name,
                detector_backend=detector_backend,
                enforce_detection=False,
                align=True,
            )
            return self._extract_embedding_from_result(embedding)
        except Exception:
            return None

    def _to_rgb_array(self, image: Image.Image) -> np.ndarray:
        img_array = np.array(image)
        if len(img_array.shape) == 2:
            img_array = cv2.cvtColor(img_array, cv2.COLOR_GRAY2RGB)
        elif img_array.shape[2] == 4:
            img_array = cv2.cvtColor(img_array, cv2.COLOR_RGBA2RGB)
        return img_array

    def _extract_face_payload(self, img_array: np.ndarray) -> Optional[Dict]:
        """
        Extract first face payload with anti-spoof info where available.
        DeepFace anti_spoofing availability varies by version; this falls back safely.
        """
        for detector_backend in self.detector_priority:
            try:
                faces = DeepFace.extract_faces(
                    img_path=img_array,
                    detector_backend=detector_backend,
                    enforce_detection=False,
                    align=True,
                    anti_spoofing=True
                )
            except TypeError:
                try:
                    faces = DeepFace.extract_faces(
                        img_path=img_array,
                        detector_backend=detector_backend,
                        enforce_detection=False,
                        align=True
                    )
                except Exception:
                    continue
            except Exception:
                continue

            if not faces:
                continue
            first = faces[0]
            if isinstance(first, dict):
                return first
        return None

    def _extract_face_bbox(self, img_array: np.ndarray, payload: Optional[Dict]) -> Optional[Tuple[int, int, int, int]]:
        if payload and isinstance(payload, dict):
            area = payload.get("facial_area")
            if isinstance(area, dict):
                x = int(area.get("x", 0))
                y = int(area.get("y", 0))
                w = int(area.get("w", 0))
                h = int(area.get("h", 0))
                if w > 0 and h > 0:
                    return (x, y, w, h)
        return None

    def _strict_face_present(self, img_array: np.ndarray) -> bool:
        """
        Strict face-presence gate to prevent non-face objects (e.g., a hand)
        from passing into liveness scoring.
        """
        for detector_backend in self.detector_priority:
            try:
                faces = DeepFace.extract_faces(
                    img_path=img_array,
                    detector_backend=detector_backend,
                    enforce_detection=True,
                    align=True,
                )
            except TypeError:
                try:
                    faces = DeepFace.extract_faces(
                        img_path=img_array,
                        detector_backend=detector_backend,
                        enforce_detection=True,
                        align=True,
                        anti_spoofing=False,
                    )
                except Exception:
                    continue
            except Exception:
                continue

            if not faces:
                continue
            first = faces[0] if isinstance(faces, list) and faces else None
            if not isinstance(first, dict):
                return True
            conf = first.get("confidence")
            if conf is None:
                return True
            try:
                if float(conf) >= 0.20:
                    return True
            except Exception:
                return True
        return False

    def _passive_quality_score(self, img_array: np.ndarray, face_bbox: Optional[Tuple[int, int, int, int]]) -> float:
        gray = cv2.cvtColor(img_array, cv2.COLOR_RGB2GRAY)

        # Blur/focus check.
        blur_score = min(1.0, cv2.Laplacian(gray, cv2.CV_64F).var() / 180.0)

        # Edge texture richness.
        edges = cv2.Canny(gray, 60, 160)
        edge_density = np.sum(edges > 0) / float(edges.shape[0] * edges.shape[1])
        edge_score = min(1.0, edge_density * 6.0)

        # Color variance (printed/photo attacks often have lower local variance).
        color_var = float(np.var(img_array.astype(np.float32)))
        color_score = min(1.0, color_var / 2500.0)

        # Highlight penalty (screen replay glare).
        hsv = cv2.cvtColor(img_array, cv2.COLOR_RGB2HSV)
        v = hsv[:, :, 2]
        glare_ratio = float(np.mean(v > 245))
        glare_score = max(0.0, 1.0 - min(1.0, glare_ratio * 6.0))

        roi_score = 1.0
        if face_bbox is not None:
            x, y, w, h = face_bbox
            x2, y2 = min(x + w, img_array.shape[1]), min(y + h, img_array.shape[0])
            if x2 > x and y2 > y:
                face_roi = gray[y:y2, x:x2]
                if face_roi.size > 0:
                    roi_blur = min(1.0, cv2.Laplacian(face_roi, cv2.CV_64F).var() / 120.0)
                    roi_score = roi_blur

        # Weighted passive quality score.
        score = (
            (0.30 * blur_score)
            + (0.22 * edge_score)
            + (0.20 * color_score)
            + (0.18 * glare_score)
            + (0.10 * roi_score)
        )
        return float(max(0.0, min(1.0, score)))

    def _exposure_score(self, img_array: np.ndarray) -> float:
        """Return an exposure quality score (0..1) to avoid false liveness rejects in dim/glare scenes."""
        hsv = cv2.cvtColor(img_array, cv2.COLOR_RGB2HSV)
        mean_v = float(np.mean(hsv[:, :, 2]))
        if mean_v <= 35.0 or mean_v >= 245.0:
            return 0.0
        if mean_v < 70.0:
            return float((mean_v - 35.0) / 35.0)
        if mean_v <= 210.0:
            return 1.0
        return float((245.0 - mean_v) / 35.0)

    def estimate_capture_quality(self, image: Image.Image) -> Tuple[float, Dict]:
        """
        Estimate image capture quality for enterprise gating before enrollment/matching.
        """
        img_array = self._to_rgb_array(image)
        payload = self._extract_face_payload(img_array)
        bbox = self._extract_face_bbox(img_array, payload)
        if bbox is None:
            return 0.0, {"reason": "no face detected"}
        passive_quality = self._passive_quality_score(img_array, bbox)
        exposure_score = self._exposure_score(img_array)
        x, y, w, h = bbox
        face_area_ratio = float((w * h) / max(1, img_array.shape[0] * img_array.shape[1]))
        size_score = min(1.0, face_area_ratio / 0.16)  # around 16% frame occupancy is ideal.
        score = (0.55 * passive_quality) + (0.25 * exposure_score) + (0.20 * size_score)
        return float(max(0.0, min(1.0, score))), {
            "passive_quality": round(passive_quality, 4),
            "exposure_score": round(exposure_score, 4),
            "size_score": round(size_score, 4),
            "face_area_ratio": round(face_area_ratio, 4),
        }

    def _eyes_open_count(self, img_array: np.ndarray, face_bbox: Optional[Tuple[int, int, int, int]]) -> int:
        gray = cv2.cvtColor(img_array, cv2.COLOR_RGB2GRAY)
        eye_cascade = cv2.CascadeClassifier(cv2.data.haarcascades + 'haarcascade_eye.xml')
        if face_bbox is None:
            eyes = eye_cascade.detectMultiScale(gray, 1.1, 6)
            return int(len(eyes))

        x, y, w, h = face_bbox
        x2, y2 = min(x + w, img_array.shape[1]), min(y + h, img_array.shape[0])
        if x2 <= x or y2 <= y:
            return 0
        roi = gray[y:y2, x:x2]
        eyes = eye_cascade.detectMultiScale(roi, 1.1, 6)
        return int(len(eyes))

    def _active_temporal_score(
        self,
        frames: List[np.ndarray],
        challenge_type: str,
    ) -> Tuple[float, Dict]:
        if len(frames) < 2:
            return 0.0, {"reason": "active liveness requires at least 2 frames"}

        boxes: List[Tuple[int, int, int, int]] = []
        centers: List[Tuple[float, float]] = []
        areas: List[float] = []
        eye_counts: List[int] = []

        anti_spoof_scores: List[float] = []
        anti_spoof_real_votes = 0

        for f in frames:
            payload = self._extract_face_payload(f)
            bbox = self._extract_face_bbox(f, payload)
            if bbox is None:
                continue
            boxes.append(bbox)
            x, y, w, h = bbox
            centers.append((x + (w / 2.0), y + (h / 2.0)))
            areas.append(float(w * h))
            eye_counts.append(self._eyes_open_count(f, bbox))

            if payload:
                score_val = payload.get("antispoof_score")
                if score_val is not None:
                    try:
                        anti_spoof_scores.append(float(score_val))
                    except Exception:
                        pass
                is_real = payload.get("is_real")
                if isinstance(is_real, bool) and is_real:
                    anti_spoof_real_votes += 1

        if len(centers) < 2:
            return 0.0, {"reason": "no stable face trajectory detected"}

        h_img, w_img = frames[0].shape[:2]
        diag = float(np.sqrt((w_img ** 2) + (h_img ** 2))) + 1e-6

        movement = 0.0
        for i in range(1, len(centers)):
            dx = centers[i][0] - centers[i - 1][0]
            dy = centers[i][1] - centers[i - 1][1]
            movement += float(np.sqrt((dx ** 2) + (dy ** 2)))
        movement_norm = min(1.0, (movement / max(1, len(centers) - 1)) / (diag * 0.08))

        area_max = max(areas) if areas else 1.0
        area_min = min(areas) if areas else 1.0
        scale_change = min(1.0, (area_max - area_min) / max(area_max, 1.0))

        horizontal_shift = abs(centers[-1][0] - centers[0][0]) / max(float(w_img), 1.0)
        horizontal_shift_score = min(1.0, horizontal_shift / 0.12)

        blink_detected = (max(eye_counts) >= 2 and min(eye_counts) <= 1) if eye_counts else False
        blink_score = 1.0 if blink_detected else 0.0

        anti_spoof_avg = float(np.mean(anti_spoof_scores)) if anti_spoof_scores else 0.5
        anti_spoof_vote_score = anti_spoof_real_votes / float(max(1, len(boxes)))

        challenge = (challenge_type or "any").lower()
        if challenge in ("move_closer", "move_away", "depth"):
            challenge_score = scale_change
        elif challenge in ("turn_left", "turn_right"):
            challenge_score = horizontal_shift_score
        elif challenge in ("blink",):
            challenge_score = blink_score
        else:
            challenge_score = max(movement_norm, max(scale_change, horizontal_shift_score))

        temporal_score = (
            (0.32 * challenge_score)
            + (0.18 * movement_norm)
            + (0.15 * scale_change)
            + (0.10 * blink_score)
            + (0.15 * anti_spoof_avg)
            + (0.10 * anti_spoof_vote_score)
        )

        details = {
            "challenge": challenge,
            "movement_score": round(movement_norm, 4),
            "scale_score": round(scale_change, 4),
            "horizontal_shift_score": round(horizontal_shift_score, 4),
            "blink_score": round(blink_score, 4),
            "challenge_score": round(challenge_score, 4),
            "anti_spoof_avg": round(anti_spoof_avg, 4),
            "anti_spoof_real_vote_ratio": round(anti_spoof_vote_score, 4),
            "usable_frames": len(boxes),
        }
        return float(max(0.0, min(1.0, temporal_score))), details
    
    async def extract_vector(self, image: Image.Image) -> Tuple[Optional[np.ndarray], float]:
        """
        Extract face vector from image using DeepFace.
        
        Args:
            image: PIL Image object
            
        Returns:
            Tuple of (vector, confidence_score) or (None, 0.0) if no face detected
        """
        vectors = await self.extract_multi_model_vectors(image, [self.model_name])
        if vectors:
            first_model = next(iter(vectors.keys()))
            vector = vectors[first_model]
            confidence = min(1.0, np.linalg.norm(vector) / 10.0)
            return vector, float(confidence)

        # Multi-model fallback for legacy compatibility.
        vectors = await self.extract_multi_model_vectors(image, self.model_priority)
        if not vectors:
            return None, 0.0
        first_model = next(iter(vectors.keys()))
        vector = vectors[first_model]
        confidence = min(1.0, np.linalg.norm(vector) / 10.0)
        return vector, float(confidence)

    async def extract_multi_model_vectors(
        self,
        image: Image.Image,
        model_candidates: Optional[List[str]] = None
    ) -> Dict[str, np.ndarray]:
        """
        Extract embeddings for multiple models with detector fallback.
        Returns map model_name -> embedding for successful models.
        """
        try:
            img_array = self._to_rgb_array(image)
            models = self._ordered_unique(model_candidates or self.model_priority)
            out: Dict[str, np.ndarray] = {}
            for model_name in models:
                for detector_backend in self.detector_priority:
                    vec = self._represent_single(img_array, model_name, detector_backend)
                    if vec is not None:
                        out[model_name] = vec
                        break
            return out
        except Exception:
            return {}
    
    async def detect_liveness(
        self,
        image: Image.Image,
        liveness_type: str = "passive",
        frames: Optional[List[Image.Image]] = None,
        challenge_type: str = "any"
    ) -> Tuple[bool, float, float, Dict]:
        """
        Detect if face is live (real person vs photo/spoof).
        
        Args:
            image: PIL Image object
            liveness_type: 'active' (requires movement) or 'passive' (texture analysis)
            
        Returns:
            Tuple of (is_live, liveness_score, confidence)
        """
        try:
            img_array = self._to_rgb_array(image)
            if not self._strict_face_present(img_array):
                return False, 0.0, 0.0, {"reason": "no face detected"}

            payload = self._extract_face_payload(img_array)
            bbox = self._extract_face_bbox(img_array, payload)

            if bbox is None:
                return False, 0.0, 0.0, {"reason": "no face detected"}

            anti_spoof_score = 0.5
            anti_spoof_vote = 0.0
            anti_spoof_supported = False
            if payload:
                score_val = payload.get("antispoof_score")
                if score_val is not None:
                    try:
                        anti_spoof_score = float(score_val)
                        anti_spoof_supported = True
                    except Exception:
                        anti_spoof_score = 0.5
                is_real = payload.get("is_real")
                if isinstance(is_real, bool):
                    anti_spoof_vote = 1.0 if is_real else 0.0
                    anti_spoof_supported = True

            passive_quality = self._passive_quality_score(img_array, bbox)
            exposure_score = self._exposure_score(img_array)
            if anti_spoof_supported:
                passive_score = (
                    (0.60 * passive_quality)
                    + (0.15 * exposure_score)
                    + (0.15 * anti_spoof_score)
                    + (0.10 * anti_spoof_vote)
                )
            else:
                # If anti-spoof metadata is unavailable on a device, rely on robust passive quality + exposure.
                passive_score = (0.78 * passive_quality) + (0.22 * exposure_score)

            if liveness_type == "passive":
                threshold = settings.PASSIVE_LIVENESS_THRESHOLD
                threshold = min(threshold, 0.56)
                if exposure_score < 0.25:
                    threshold = min(0.66, threshold + 0.02)

                if not anti_spoof_supported:
                    threshold = min(threshold, 0.53)
                    if passive_quality >= 0.58 and exposure_score >= 0.25:
                        is_live = True
                    else:
                        is_live = passive_score >= threshold
                else:
                    strong_spoof_signal = (
                        anti_spoof_vote <= 0.0
                        and anti_spoof_score < 0.15
                        and passive_quality < 0.62
                    )
                    if strong_spoof_signal:
                        is_live = False
                    elif passive_quality >= 0.64 and exposure_score >= 0.35:
                        is_live = True
                    else:
                        is_live = passive_score >= threshold
                details = {
                    "passive_quality": round(passive_quality, 4),
                    "exposure_score": round(exposure_score, 4),
                    "anti_spoof_score": round(anti_spoof_score, 4),
                    "anti_spoof_vote": round(anti_spoof_vote, 4),
                    "anti_spoof_supported": anti_spoof_supported,
                    "threshold": round(threshold, 4),
                }
                return bool(is_live), float(passive_score), float(passive_score), details

            if liveness_type == "active":
                frame_arrays: List[np.ndarray] = [img_array]
                if frames:
                    for frame in frames:
                        frame_arrays.append(self._to_rgb_array(frame))
                temporal_score, temporal_details = self._active_temporal_score(frame_arrays, challenge_type)

                final_score = (0.45 * passive_score) + (0.55 * temporal_score)
                threshold = settings.ACTIVE_LIVENESS_THRESHOLD
                if not anti_spoof_supported:
                    threshold = min(threshold, 0.66)
                if int(temporal_details.get("usable_frames", 0)) >= 3:
                    threshold -= 0.03
                threshold = max(0.50, min(threshold, 0.70))

                passive_guard = max(0.45, min(settings.PASSIVE_LIVENESS_THRESHOLD*0.82, 0.56))
                movement_score = float(temporal_details.get("movement_score", 0.0))
                challenge_score = float(temporal_details.get("challenge_score", 0.0))
                # Active liveness must demonstrate real temporal variation.
                temporal_gate = movement_score >= 0.07 or challenge_score >= 0.16
                is_live = final_score >= threshold and passive_score >= passive_guard and temporal_gate
                if not is_live and temporal_score >= 0.72 and passive_quality >= 0.58:
                    # Accept high-confidence temporal movement when passive signal is reasonable.
                    is_live = True

                reason = None
                if not is_live:
                    if str(temporal_details.get("reason", "")).strip():
                        reason = str(temporal_details.get("reason"))
                    elif not temporal_gate:
                        reason = "insufficient temporal movement"
                    elif passive_score < passive_guard:
                        reason = "passive guard not met"
                    elif final_score < threshold:
                        reason = "active score below threshold"

                details = {
                    "passive_score": round(passive_score, 4),
                    "passive_quality": round(passive_quality, 4),
                    "exposure_score": round(exposure_score, 4),
                    "temporal_score": round(temporal_score, 4),
                    "passive_guard": round(passive_guard, 4),
                    "threshold": round(threshold, 4),
                    "challenge_type": challenge_type,
                    **temporal_details,
                }
                if reason:
                    details["reason"] = reason
                return bool(is_live), float(final_score), float(final_score), details

            else:
                raise ValueError(f"Unknown liveness type: {liveness_type}")
                
        except Exception as e:
            # On error, assume not live (fail secure)
            return False, 0.0, 0.0, {"error": str(e)}
    
    async def detect_face(self, image: Image.Image) -> Optional[Tuple[int, int, int, int]]:
        """
        Detect face bounding box in image.
        
        Returns:
            Bounding box as (x, y, width, height) or None
        """
        try:
            img_array = np.array(image)
            
            if len(img_array.shape) == 2:
                img_array = cv2.cvtColor(img_array, cv2.COLOR_GRAY2RGB)
            elif img_array.shape[2] == 4:
                img_array = cv2.cvtColor(img_array, cv2.COLOR_RGBA2RGB)
            
            # Use DeepFace to detect face
            face_objs = DeepFace.extract_faces(
                img_path=img_array,
                detector_backend=self.detector,
                enforce_detection=False
            )
            
            if not face_objs or len(face_objs) == 0:
                return None
            
            # Return first face bounding box
            # Note: DeepFace doesn't directly return bbox, so we use OpenCV
            face_cascade = cv2.CascadeClassifier(cv2.data.haarcascades + 'haarcascade_frontalface_default.xml')
            gray = cv2.cvtColor(img_array, cv2.COLOR_RGB2GRAY)
            faces = face_cascade.detectMultiScale(gray, 1.1, 4)
            
            if len(faces) > 0:
                x, y, w, h = faces[0]
                return (int(x), int(y), int(w), int(h))
            
            return None
            
        except Exception as e:
            return None
