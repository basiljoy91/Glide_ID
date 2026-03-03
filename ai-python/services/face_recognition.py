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
        self.model_name = settings.DEEPFACE_MODEL
        self.backend = settings.DEEPFACE_BACKEND
        self.detector = settings.DEEPFACE_DETECTOR
        
        # Suppress TensorFlow warnings
        tf.get_logger().setLevel('ERROR')
        os.environ['TF_CPP_MIN_LOG_LEVEL'] = '2'

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
        try:
            faces = DeepFace.extract_faces(
                img_path=img_array,
                detector_backend=self.detector,
                enforce_detection=False,
                align=True,
                anti_spoofing=True
            )
        except TypeError:
            faces = DeepFace.extract_faces(
                img_path=img_array,
                detector_backend=self.detector,
                enforce_detection=False,
                align=True
            )
        except Exception:
            return None

        if not faces:
            return None
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

        gray = cv2.cvtColor(img_array, cv2.COLOR_RGB2GRAY)
        face_cascade = cv2.CascadeClassifier(cv2.data.haarcascades + 'haarcascade_frontalface_default.xml')
        faces = face_cascade.detectMultiScale(gray, 1.1, 4)
        if len(faces) == 0:
            return None
        x, y, w, h = faces[0]
        return (int(x), int(y), int(w), int(h))

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
        try:
            # Convert PIL Image to numpy array
            img_array = np.array(image)
            
            # Ensure image is in RGB format
            if len(img_array.shape) == 2:  # Grayscale
                img_array = cv2.cvtColor(img_array, cv2.COLOR_GRAY2RGB)
            elif img_array.shape[2] == 4:  # RGBA
                img_array = cv2.cvtColor(img_array, cv2.COLOR_RGBA2RGB)
            
            def _extract_from_result(result) -> Optional[np.ndarray]:
                if not result:
                    return None
                if isinstance(result, list):
                    payload = result[0] if result else None
                elif isinstance(result, dict):
                    payload = result
                else:
                    payload = None

                if not payload or 'embedding' not in payload:
                    return None
                return np.array(payload['embedding'], dtype=np.float32)

            vector: Optional[np.ndarray] = None
            try:
                embedding = DeepFace.represent(
                    img_path=img_array,
                    model_name=self.model_name,
                    detector_backend=self.detector,
                    enforce_detection=True,
                    align=True
                )
                vector = _extract_from_result(embedding)
            except Exception as detection_error:
                detection_message = str(detection_error).lower()
                no_face_signals = [
                    "face could not be detected",
                    "face cannot be detected",
                    "could not detect face",
                    "enforce_detection"
                ]
                if any(signal in detection_message for signal in no_face_signals):
                    # Fallback: allow DeepFace to derive an embedding from the best-effort crop
                    embedding = DeepFace.represent(
                        img_path=img_array,
                        model_name=self.model_name,
                        detector_backend=self.detector,
                        enforce_detection=False,
                        align=True
                    )
                    vector = _extract_from_result(embedding)
                else:
                    raise

            if vector is None:
                return None, 0.0
            
            # Calculate confidence (normalize to 0-1)
            # DeepFace doesn't provide explicit confidence, so we use a heuristic
            confidence = min(1.0, np.linalg.norm(vector) / 10.0)
            
            return vector, float(confidence)
            
        except ValueError as e:
            # No face detected
            if "face could not be detected" in str(e).lower():
                return None, 0.0
            raise
        except Exception as e:
            if "face could not be detected" in str(e).lower():
                return None, 0.0
            raise Exception(f"Face extraction failed: {str(e)}")
    
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
            passive_score = (0.70 * passive_quality) + (0.20 * anti_spoof_score) + (0.10 * anti_spoof_vote)

            if liveness_type == "passive":
                threshold = settings.PASSIVE_LIVENESS_THRESHOLD
                # Many camera/device combos do not expose reliable anti-spoof metadata;
                # when unavailable, apply a relaxed threshold to reduce false rejects.
                if not anti_spoof_supported:
                    threshold = min(threshold, 0.62)
                # Secondary guardrail: allow strong passive quality even without anti-spoof metadata.
                if not anti_spoof_supported and passive_quality >= 0.66:
                    is_live = True
                else:
                    is_live = passive_score >= threshold
                details = {
                    "passive_quality": round(passive_quality, 4),
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
                    threshold = min(threshold, 0.72)
                is_live = final_score >= threshold and passive_score >= (settings.PASSIVE_LIVENESS_THRESHOLD * 0.85)

                details = {
                    "passive_score": round(passive_score, 4),
                    "temporal_score": round(temporal_score, 4),
                    "threshold": round(threshold, 4),
                    "challenge_type": challenge_type,
                    **temporal_details,
                }
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
